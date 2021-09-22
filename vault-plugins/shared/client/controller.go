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

var (
	ErrNotSetConf = fmt.Errorf("access configuration does not set")
	ErrNotInit    = fmt.Errorf("client not init")
)

type VaultClientController struct {
	clientLock sync.RWMutex
	apiClient  *api.Client

	accessConfigStorage *accessConfigStorage
	// accessConfigStorageFactory func(s logical.Storage) *accessConfigStorage
	loggerFactory func() log.Logger
}

// NewVaultClientController returns uninitialized VaultClientController
func NewVaultClientController(loggerFactory func() log.Logger, storage logical.Storage) *VaultClientController {
	c := &VaultClientController{
		loggerFactory:       loggerFactory,
		accessConfigStorage: newAccessConfigStorage(storage),
	}
	// c.accessConfigStorageFactory = newAccessConfigStorage

	return c
}

// GetApiConfig get vault api access config (APIURL, APIHost, APICa)
// if configuration not found returns nil pointer
func (c *VaultClientController) GetApiConfig(ctx context.Context) (*VaultApiConf, error) {
	conf, err := c.accessConfigStorage.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if conf == nil {
		return nil, nil
	}

	return &VaultApiConf{
		APIURL:  conf.APIURL,
		APIHost: conf.APIHost,
		APICa:   conf.APICa,
	}, nil
}

// init initialize api client by demand
// if store don't contains configuration it may return ErrNotSetConf error
// it is normal case for just started and not configured plugin
func (c *VaultClientController) init() error {
	logger := c.loggerFactory().Named("init")
	logger.Debug("started")
	defer logger.Debug("exit")
	c.clientLock.RLock()
	defer c.clientLock.RUnlock()
	if c.apiClient != nil {
		return nil
	}

	ctx := context.Background()
	curConf, err := c.accessConfigStorage.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("getting access config: %w", err)
	}

	if curConf == nil {
		return ErrNotSetConf
	}

	apiClient, err := newAPIClient(curConf)
	if err != nil {
		return fmt.Errorf("creating api client: %w", err)
	}

	appRole := newAccessClient(apiClient, curConf, c.loggerFactory()).AppRole()

	auth, err := appRole.Login()
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	apiClient.SetToken(auth.ClientToken)
	c.setAPIClient(apiClient)
	logger.Debug("normal finish")
	return nil
}

// APIClient getting vault api client for communicate between plugins and vault
func (c *VaultClientController) APIClient() (*api.Client, error) {
	c.clientLock.RLock()
	apiClient := c.apiClient
	c.clientLock.RUnlock()
	if apiClient == nil {
		err := c.init()
		if err != nil {
			return nil, err
		}
		c.clientLock.RLock()
		apiClient = c.apiClient
		c.clientLock.RUnlock()

	}
	return apiClient, nil
}

// ReInit full reinitialisation apiClient
// don't need to use in regular code. Use Init
func (c *VaultClientController) ReInit(s logical.Storage) error {
	// clear client
	c.setAPIClient(nil)
	// init from store
	c.accessConfigStorage = newAccessConfigStorage(s)
	return c.init()
}

func (c *VaultClientController) renewLease() error {
	c.loggerFactory().Info("Run renew lease")
	clientAPI, err := c.APIClient()
	if err != nil {
		return err
	}

	err = prolongAccessToken(clientAPI, 120)
	if err != nil {
		c.loggerFactory().Info(" prolong")
	}
	return err
}

// OnPeriodical must be called in periodical function
// if store don't contains configuration it may return ErrNotSetConf error
// it is normal case
func (c *VaultClientController) OnPeriodical(ctx context.Context, r *logical.Request) error {
	logger := c.loggerFactory()

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

	store := newAccessConfigStorage(r.Storage)

	accessConf, err := store.GetConfig(ctx)
	if err != nil {
		return err
	}

	if accessConf == nil {
		logger.Info("access not configured")
		return nil
	}

	if need, remain := accessConf.IsNeedToRenewSecretID(time.Now()); !need {
		logger.Info(fmt.Sprintf("no need renew secret id. remain %vs", remain))
		return nil
	}

	// login in with new secret id in gen function
	err = genNewSecretID(ctx, apiClient, store, accessConf, c.loggerFactory())
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
	err = genNewSecretID(ctx, apiClient, store, curConf, c.loggerFactory())
	if err != nil {
		return err
	}

	return nil
}
