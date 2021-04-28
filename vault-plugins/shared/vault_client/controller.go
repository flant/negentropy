package vault_client

import (
	"context"
	"errors"
	"fmt"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
	"sync"
	"time"
)

var NotSetConfError = fmt.Errorf("access configuration does not set")
var ClientNotInitError = fmt.Errorf("client not init")

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

	c.storageFactory = func(s logical.Storage) *accessConfigStorage {
		return &accessConfigStorage{
			parent: s,
		}
	}

	return c
}

// Init initialize api client
// if store don't contains configuration it may return NotSetConfError error
// it is normal case
func (c *VaultClientController) Init(s logical.Storage) error {
	apiClient, err := c.ApiClient()
	if err == nil && apiClient != nil {
		return nil
	}

	ctx := context.Background()
	curConf, err := c.storageFactory(s).Get(ctx)
	if err != nil {
		return err
	}

	if curConf == nil {
		return NotSetConfError
	}

	apiClient, err = newApiClient(curConf)
	if err != nil {
		return err
	}

	accessClient := newAccessClient(apiClient, curConf, c.getLogger())
	auth, err := accessClient.AppRole().Login()

	if err != nil {
		return err
	}

	apiClient.SetToken(auth.ClientToken)
	c.setApiClient(apiClient)

	return nil
}

// ApiClient getting vault api client for communicate between plugins and vault
func (c *VaultClientController) ApiClient() (*api.Client, error) {
	c.clientLock.RLock()
	defer c.clientLock.RUnlock()

	if c.apiClient == nil {
		return nil, ClientNotInitError
	}

	return c.apiClient, nil
}

// RenewSecretId must be called in periodical function
// if store don't contains configuration it may return NotSetConfError error
// it is normal case
func (c *VaultClientController) RenewSecretId(ctx context.Context, r *logical.Request) error {
	logger := c.getLogger()

	store := c.storageFactory(r.Storage)

	accessConf, err := store.Get(ctx)
	if err != nil {
		return err
	}

	if accessConf == nil {
		logger.Info("access not configurated")
		return nil
	}

	if need, remain := accessConf.IsNeedToRenewSecretId(time.Now()); !need {
		logger.Info(fmt.Sprintf("no need renew secret id. remain %vs", remain))
		return nil
	}

	apiClient, err := c.ApiClient()
	if err != nil && errors.Is(err, ClientNotInitError) {
		logger.Info("not init client nothing to renew")
		return nil
	}

	// login in with new secret id in gen function
	err = genNewSecretId(ctx, apiClient, store, accessConf, c.getLogger())
	if err != nil {
		return err
	}

	logger.Info("secret id renewed")

	return nil
}

func (c *VaultClientController) setApiClient(newClient *api.Client) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()

	c.apiClient = newClient
}

func (c *VaultClientController) setAccessConfig(ctx context.Context, store *accessConfigStorage, curConf *vaultAccessConfig) error {
	var err error
	apiClient, err := c.ApiClient()
	if errors.Is(err, ClientNotInitError) || apiClient == nil {
		apiClient, err = newApiClient(curConf)
		if err != nil {
			return err
		}

		defer func() {
			if err == nil {
				c.setApiClient(apiClient)
			}
		}()
	}

	// login in with new secret id in gen function
	err = genNewSecretId(ctx, apiClient, store, curConf, c.getLogger())
	if err != nil {
		return err
	}

	return nil
}
