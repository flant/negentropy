// VaultClientController implements AccessVaultClientController and has the following logic inside:
// Possible states:
// 1) there is now configuration in storage. All methods (except HandleConfigureVaultAccess and UpdateOutdated) returns error:
// ErrNotSetConf
// 2) there is a configuration in storage and a valid client, ready for using.
// 3) there is a configuration in storage and invalid client
// From 1st state to 2nd controller goes during successful call of  HandleConfigureVaultAccess.
// Any changes of role_id/secret_id/token - leads to updating client
// From 2nd to 3rd - is an accident - vault was shouted down, and the token in client is outdated,
// From 3rd to 2nd - by running UpdateOutdated, or during call constructor (NewAccessVaultClientController)

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
	// GetApiConfig returns config for using in read config path;
	// just returns stored config
	GetApiConfig(context.Context) (*VaultApiConf, error)
	// APIClient returns prepared client or error;
	// just return active client
	APIClient() (*api.Client, error)
	// UpdateOutdated runs periodical function for support workable state;
	// take active client, update all obsolescent token etc, update configuration, update client
	UpdateOutdated(context.Context) error
	// HandleConfigureVaultAccess serves requests for configuration;
	// 1) make client based on given role_id/secret_id,
	// 2) recreate role_id/secret_id,
	// 3) make client based on new  role_id/secret_id,
	// 4) store config
	// 5) store client
	HandleConfigureVaultAccess(context.Context, *logical.Request, *framework.FieldData) (*logical.Response, error)
}

type VaultClientController struct {
	storage logical.Storage

	clientLock sync.RWMutex
	apiClient  *api.Client

	logger hclog.Logger
}

// NewAccessVaultClientController returns uninitialized VaultClientController
func NewAccessVaultClientController(storage logical.Storage, parentLogger hclog.Logger) (AccessVaultClientController, error) {
	if storage == nil {
		return nil, fmt.Errorf("storage: %w", consts.ErrNilPointer)
	}
	c := &VaultClientController{
		logger:  parentLogger.Named("ApiClientController"),
		storage: storage,
	}
	// TODO get config and create client
	return c, nil
}

// GetApiConfig get vault api access config (APIURL, APIHost, CaCert)
// if configuration not found returns nil pointer
func (c *VaultClientController) GetApiConfig(ctx context.Context) (*VaultApiConf, error) {
	conf, err := c.getVaultClientConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &VaultApiConf{
		APIURL:  conf.APIURL,
		APIHost: conf.APIHost,
		CaCert:  conf.CaCert,
	}, nil
}

// Init initialize api client by demand
// if store don't contain configuration it may return ErrNotSetConf error
// it is normal case for just started and not configured plugin
func (c *VaultClientController) Init() (*api.Client, error) {
	logger := c.logger.Named("init")
	logger.Debug("started")
	defer logger.Debug("exit")

	ctx := context.Background()
	curConf, err := c.getVaultClientConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting access config: %w", err)
	}

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
	c.clientLock.RLock()
	apiClient := c.apiClient
	c.clientLock.RUnlock()

	if apiClient == nil {
		var err error
		c.clientLock.Lock()
		defer c.clientLock.Unlock()
		apiClient, err = c.Init()
		if err != nil {
			return nil, fmt.Errorf("init and get apiClient: %w", err)
		}
	}
	return apiClient, nil
}

func (c *VaultClientController) renewToken(ctx context.Context) error {
	c.logger.Info("renew vault access token")
	accessConf, err := c.getVaultClientConfig(ctx)
	if err != nil {
		return err
	}

	err = loginAndSetToken(c.apiClient, accessConf, c.logger)
	if err != nil {
		return err
	}
	c.logger.Info("vault access token is renewed")
	return nil
}

// UpdateOutdated must be called in periodical function
// if storage don't contain configuration it warns and return
func (c *VaultClientController) UpdateOutdated(ctx context.Context) error {
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
	err = c.renewToken(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("not prolong lease %v", err))
		return err
	} else {
		logger.Info("token prolong success")
	}

	accessConf, err := c.getVaultClientConfig(ctx)
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
	err = c.genNewSecretID(ctx, apiClient, c.storage, accessConf, c.logger)
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
	err = c.genNewSecretID(ctx, apiClient, c.storage, curConf, c.logger)
	if err != nil {
		return fmt.Errorf("error replacing secret-id: %w", err)
	}

	return nil
}
