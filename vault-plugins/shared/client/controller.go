package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
)

var ErrNotSetConf = fmt.Errorf("access configuration does not set")
var ErrNotInit = fmt.Errorf("client not init")

type VaultClientController struct {
	clientLock sync.RWMutex
	apiClient  *api.Client

	storageFactory func(s logical.Storage) *accessConfigStorage
	getLogger      func() log.Logger
}

func NewVaultClientController(loggerFactory func() log.Logger) *VaultClientController {
	c := &VaultClientController{
		getLogger: loggerFactory,
	}

	c.storageFactory = newAccessConfigStorage

	return c
}

// Init initialize api client
// if store don't contains configuration it may return ErrNotSetConf error
// it is normal case
func (c *VaultClientController) Init(s logical.Storage) error {
	apiClient, err := c.APIClient()
	if err == nil && apiClient != nil {
		return nil
	}

	ctx := context.Background()
	curConf, err := c.storageFactory(s).Get(ctx)
	if err != nil {
		return err
	}

	if curConf == nil {
		return ErrNotSetConf
	}

	apiClient, err = newAPIClient(curConf)
	if err != nil {
		return err
	}

	accessClient := newAccessClient(apiClient, curConf, c.getLogger())
	auth, err := accessClient.AppRole().Login()

	if err != nil {
		return err
	}

	apiClient.SetToken(auth.ClientToken)
	c.setAPIClient(apiClient)

	return nil
}

// APIClient getting vault api client for communicate between plugins and vault
func (c *VaultClientController) APIClient() (*api.Client, error) {
	c.clientLock.RLock()
	defer c.clientLock.RUnlock()

	if c.apiClient == nil {
		return nil, ErrNotInit
	}

	return c.apiClient, nil
}

// ReInit full reinitialisation apiClient
// don't need to use in regular code. Use Init
func (c *VaultClientController) ReInit(s logical.Storage) error {
	// clear client
	c.setAPIClient(nil)
	// init from store
	return c.Init(s)
}

func (c *VaultClientController) renewLease() error {
	c.getLogger().Info("Run renew lease")
	clientAPI, err := c.APIClient()
	if err != nil {
		return err
	}

	err = prolongAccessToken(clientAPI, 120)
	if err != nil {
		c.getLogger().Info(" prolong")
	}
	return err
}

// OnPeriodical must be called in periodical function
// if store don't contains configuration it may return ErrNotSetConf error
// it is normal case
func (c *VaultClientController) OnPeriodical(ctx context.Context, r *logical.Request) error {
	logger := c.getLogger()

	apiClient, err := c.APIClient()
	if err != nil && errors.Is(err, ErrNotInit) {
		logger.Info("not init client nothing to renew")
		return nil
	}

	// always renew current token
	err = c.renewLease()
	if err != nil {
		logger.Error(fmt.Sprintf("not prolong lease %v", err))
	} else {
		logger.Info("token prolong success")
	}

	store := c.storageFactory(r.Storage)

	accessConf, err := store.Get(ctx)
	if err != nil {
		return err
	}

	if accessConf == nil {
		logger.Info("access not configurated")
		return nil
	}

	if need, remain := accessConf.IsNeedToRenewSecretID(time.Now()); !need {
		logger.Info(fmt.Sprintf("no need renew secret id. remain %vs", remain))
		return nil
	}

	// login in with new secret id in gen function
	err = genNewSecretID(ctx, apiClient, store, accessConf, c.getLogger())
	if err != nil {
		return err
	}

	logger.Info("secret id renewed")

	return nil
}

func (c *VaultClientController) setAPIClient(newClient *api.Client) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	c.apiClient = newClient
}

func (c *VaultClientController) setAccessConfig(ctx context.Context, store *accessConfigStorage, curConf *vaultAccessConfig) error {
	var err error
	apiClient, err := c.APIClient()
	if errors.Is(err, ErrNotInit) || apiClient == nil {
		apiClient, err = newAPIClient(curConf)
		if err != nil {
			return err
		}

		defer func() {
			if err == nil {
				c.setAPIClient(apiClient)
			}
		}()
	}

	// login in with new secret id in gen function
	err = genNewSecretID(ctx, apiClient, store, curConf, c.getLogger())
	if err != nil {
		return err
	}

	return nil
}
