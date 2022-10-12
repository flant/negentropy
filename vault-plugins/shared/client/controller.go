package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

var ErrNotSetConf = fmt.Errorf("%w:vault access configuration does not set", consts.ErrNotConfigured)

type AccessVaultClientController interface {
	// GetApiConfig returns config for using in read config path
	GetApiConfig(context.Context) (*VaultApiConf, error)
	// APIClient returns prepaered client or error
	APIClient() (*api.Client, error)
	// OnPeriodical runs periodical function for support workable state
	OnPeriodical(context.Context) error
	// HandleConfigureVaultAccess serves requests for configuration
	HandleConfigureVaultAccess(context.Context, *logical.Request, *framework.FieldData) (*logical.Response, error)
}

type VaultClientController struct {
	storage logical.Storage
	cfg     *vaultAccessConfig

	clientLock sync.RWMutex
	apiClient  *api.Client

	logger hclog.Logger
}

// NewAccessVaultClientController returns uninitialized VaultClientController
func NewAccessVaultClientController(storage logical.Storage, parentLogger hclog.Logger) (AccessVaultClientController, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage: %w", consts.ErrNilPointer)
	}
	return &VaultClientController{
		logger:  parentLogger.Named("ApiClientController"),
		storage: storage,
	}, nil
}

// GetApiConfig get vault api access config (APIURL, APIHost, CaCert)
// if configuration not found returns nil pointer
func (c *VaultClientController) GetApiConfig(ctx context.Context) (*VaultApiConf, error) {
	conf, err := GetVaultClientConfig(ctx, c.storage)
	if err != nil {
		return nil, err
	}

	if conf == nil {
		return nil, nil
	}
	if conf.Preferable(c.cfg) {
		c.clientLock.Lock()
		defer c.clientLock.Unlock()
		_, err := c.Init()
		if err != nil {
			return nil, err
		}
	}
	return &VaultApiConf{
		APIURL:  conf.APIURL,
		APIHost: conf.APIHost,
		CaCert:  conf.CaCert,
	}, nil
}

// init initialize api client by demand
// if store don't contain configuration it may return ErrNotSetConf error
// it is normal case for just started and not configured plugin
func (c *VaultClientController) Init() (*api.Client, error) {
	logger := c.logger.Named("init")
	logger.Debug("started")
	defer logger.Debug("exit")

	ctx := context.Background()
	curConf, err := GetVaultClientConfig(ctx, c.storage)
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
// can be tried to get vault api client, by passing storage=nil
func (c *VaultClientController) APIClient() (*api.Client, error) {
	ctx := context.Background()
	curConf, err := GetVaultClientConfig(ctx, c.storage)
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
		apiClient, err = c.Init()
		if err != nil {
			return nil, fmt.Errorf("init and get apiClient: %w", err)
		}
	}
	return apiClient, nil
}

func (c *VaultClientController) renewToken(storage logical.Storage) error {
	c.logger.Info("renew vault access token")

	err := loginAndSetToken(c.apiClient, c.cfg, c.logger)
	if err != nil {
		return err
	}
	c.logger.Info("vault access token is renewed")
	return nil
}

// OnPeriodical must be called in periodical function
// if store don't contain configuration it may return ErrNotSetConf error
// it is normal case
func (c *VaultClientController) OnPeriodical(ctx context.Context) error {
	logger := c.logger.Named("renew")
	apiClient, err := c.APIClient()
	if errors.Is(err, ErrNotSetConf) {
		logger.Warn("not init client nothing to renew")
		return nil
	}
	if err != nil {
		logger.Error(fmt.Sprintf("getting apiClient:%v", err))
		return err
	}

	// always renew current token
	err = c.renewToken(c.storage)
	if err != nil {
		logger.Error(fmt.Sprintf("not prolong lease %v", err))
		return err
	} else {
		logger.Info("token prolong success")
	}

	accessConf, err := GetVaultClientConfig(ctx, c.storage)
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
	err = genNewSecretID(ctx, apiClient, c.storage, accessConf, c.logger)
	if err != nil {
		return fmt.Errorf("genNewSecretID:%w", err)
	}

	logger.Info("secretID&token renewed")

	return nil
}

func (c *VaultClientController) setAccessConfig(ctx context.Context, curConf *vaultAccessConfig) error {
	var err error
	apiClient, err := c.APIClient()
	if errors.Is(err, ErrNotSetConf) || apiClient == nil {
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
	err = genNewSecretID(ctx, apiClient, c.storage, curConf, c.logger)
	if err != nil {
		return fmt.Errorf("error replacing secret-id: %w", err)
	}

	return nil
}
