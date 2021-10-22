package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
)

var (
	ErrNotSetConf = fmt.Errorf("access configuration does not set")
	ErrNotInit    = fmt.Errorf("client not init")
)

type VaultClientController struct {
	cfg *vaultAccessConfig

	clientLock sync.RWMutex
	apiClient  *api.Client

	logger hclog.Logger
}

// NewVaultClientController returns uninitialized VaultClientController
func NewVaultClientController(parentLogger hclog.Logger) *VaultClientController {
	c := &VaultClientController{
		logger: parentLogger.Named("ApiClient"),
	}
	return c
}

// GetApiConfig get vault api access config (APIURL, APIHost, APICa)
// if configuration not found returns nil pointer
func (c *VaultClientController) GetApiConfig(ctx context.Context, storage logical.Storage) (*VaultApiConf, error) {
	accessConfigStorage := newAccessConfigStorage(storage)
	conf, err := accessConfigStorage.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if conf == nil {
		return nil, nil
	}
	if conf.Preferable(c.cfg) {
		c.clientLock.Lock()
		defer c.clientLock.Unlock()
		_, err := c.init(storage)
		if err != nil {
			return nil, err
		}
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
func (c *VaultClientController) init(storage logical.Storage) (*api.Client, error) {
	logger := c.logger.Named("init")
	logger.Debug("started")
	defer logger.Debug("exit")

	ctx := context.Background()
	accessConfigStorage := newAccessConfigStorage(storage)
	curConf, err := accessConfigStorage.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting access config: %w", err)
	}

	if curConf == nil {
		return nil, ErrNotSetConf
	}

	c.cfg = curConf

	apiClient, err := newAPIClient(curConf)
	if err != nil {
		return nil, fmt.Errorf("creating api client: %w", err)
	}

	appRole := newAccessClient(apiClient, curConf, c.logger).AppRole()

	auth, err := appRole.Login()
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}

	apiClient.SetToken(auth.ClientToken)
	c.apiClient = apiClient
	logger.Debug("normal finish")
	return apiClient, nil
}

// APIClient getting vault api client for communicate between plugins and vault
func (c *VaultClientController) APIClient(storage logical.Storage) (*api.Client, error) {
	ctx := context.Background()
	accessConfigStorage := newAccessConfigStorage(storage)
	curConf, err := accessConfigStorage.GetConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting access config: %w", err)
	}

	c.clientLock.RLock()
	apiClient := c.apiClient
	c.clientLock.RUnlock()

	if apiClient == nil ||
		curConf.Preferable(c.cfg) {
		c.clientLock.Lock()
		defer c.clientLock.Unlock()
		apiClient, err = c.init(storage)
		if err != nil {
			return nil, fmt.Errorf("init and get apiClient: %w", err)
		}
	}
	return apiClient, nil
}

// ReInit full reinitialisation apiClient
// don't need to use in regular code. Use Init
func (c *VaultClientController) ReInit(storage logical.Storage) error {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	c.apiClient = nil
	_, err := c.init(storage)
	return err
}

// func (c *VaultClientController) renewLease(storage logical.Storage) error {
//	c.logger.Info("Run renew lease")
//	clientAPI, err := c.APIClient(storage)
//	if err != nil {
//		return err
//	}
//
//	err = prolongAccessToken(clientAPI, 120, c.logger)
//	if err != nil {
//		return err
//	}
//	c.logger.Info(" prolong")
//	return nil
// }

// OnPeriodical must be called in periodical function
// if store don't contains configuration it may return ErrNotSetConf error
// it is normal case
func (c *VaultClientController) OnPeriodical(ctx context.Context, r *logical.Request) error {
	logger := c.logger.Named("renew")
	apiClient, err := c.APIClient(r.Storage)
	if err != nil && errors.Is(err, ErrNotInit) {
		logger.Warn("not init client nothing to renew")
		return nil
	}

	// always renew current token
	// err = c.renewLease(r.Storage)
	// if err != nil {
	//	logger.Error(fmt.Sprintf("not prolong lease %v", err))
	// } else {
	//	logger.Info("token prolong success")
	// }

	store := newAccessConfigStorage(r.Storage)

	accessConf, err := store.GetConfig(ctx)
	if err != nil {
		return err
	}

	if accessConf == nil {
		logger.Warn("access not configured")
		return nil
	}

	if need, remain := accessConf.IsNeedToRenewSecretID(time.Now()); !need {
		logger.Info(fmt.Sprintf("no need renew secret id. remain %vs", remain))
		return nil
	}

	// login in with new secret id in gen function
	logger.Debug("try renew secretID&token")
	err = genNewSecretID(ctx, apiClient, store, accessConf, c.logger)
	if err != nil {
		return fmt.Errorf("genNewSecretID:%w", err)
	}

	logger.Info("secretID&token renewed")

	return nil
}

func (c *VaultClientController) setAccessConfig(ctx context.Context, storage logical.Storage, curConf *vaultAccessConfig) error {
	var err error
	apiClient, err := c.APIClient(storage)
	if errors.Is(err, ErrNotInit) || apiClient == nil {
		apiClient, err = newAPIClient(curConf)
		if err != nil {
			return err
		}

		defer func() {
			if err == nil {
				c.clientLock.Lock()
				defer c.clientLock.Unlock()
				c.apiClient = apiClient
			}
		}()
	}

	// login in with new secret id in gen function and save config to storage
	accessConfigStorage := newAccessConfigStorage(storage)
	err = genNewSecretID(ctx, apiClient, accessConfigStorage, curConf, c.logger)
	if err != nil {
		return err
	}

	return nil
}
