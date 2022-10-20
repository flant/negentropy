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

	// protect apiClient and configuration from races
	mutex     sync.RWMutex
	apiClient *api.Client

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
	conf, err := getVaultClientConfig(context.Background(), c.storage)
	if errors.Is(err, ErrNotSetConf) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	err = c.initClient(conf) // initialize apiClient
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetApiConfig get vault api access config (APIURL, APIHost, CaCert)
// if configuration not found returns nil pointer
func (c *VaultClientController) GetApiConfig(ctx context.Context) (*VaultApiConf, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	conf, err := getVaultClientConfig(ctx, c.storage)
	if err != nil {
		return nil, err
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
func (c *VaultClientController) initClient(conf *vaultAccessConfig) error {
	logger := c.logger.Named("init")
	logger.Debug("started")
	defer logger.Debug("exit")

	apiClient, err := conf.newAPIClient()
	if err != nil {
		return fmt.Errorf("creating api client: %w", err)
	}

	err = c.loginByApproleAndUpdateToken(apiClient, conf)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	c.apiClient = apiClient
	logger.Debug("normal finish")
	return nil
}

// APIClient getting vault api client for communicate between plugins and vault
// can be tried to get vault api client, by passing storage=nil
func (c *VaultClientController) APIClient() (*api.Client, error) {
	c.mutex.RLock()
	apiClient := c.apiClient
	c.mutex.RUnlock()
	if apiClient == nil {
		return nil, ErrNotSetConf
	}
	clientCopy, err := api.NewClient(apiClient.CloneConfig())
	if err != nil {
		return nil, err
	}
	clientCopy.SetToken(apiClient.Token())
	return clientCopy, nil
}

// UpdateOutdated must be called in periodical function
// if storage don't contain configuration it warns and return
func (c *VaultClientController) UpdateOutdated(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	logger := c.logger.Named("renew")
	apiClient := c.apiClient
	if apiClient == nil {
		logger.Warn("not init client nothing to renew")
		return nil
	}

	// always renew current token
	err := c.renewToken(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("not prolong lease %v", err))
		return err
	} else {
		logger.Info("token prolong success")
	}

	conf, err := getVaultClientConfig(ctx, c.storage)
	if err != nil {
		return err
	}

	if conf == nil {
		logger.Warn("access not configured")
		return nil
	}

	if need, remain := conf.IsNeedToRenewSecretID(time.Now()); !need {
		logger.Info(fmt.Sprintf("no need renew secret id. remain %vs", remain))
		return nil
	}

	// login in with new secret id in gen function
	logger.Debug("try renew secretID&token")
	err = c.renewSecretID(ctx, apiClient, conf)
	if err != nil {
		return fmt.Errorf("renewSecretID:%w", err)
	}

	logger.Info("secretID&token renewed")

	return nil
}

func (c *VaultClientController) setAccessConfig(ctx context.Context, conf *vaultAccessConfig) error {
	err := c.initClient(conf)
	if err != nil {
		return err
	}
	// login in with new secret id in gen function and save config to storage
	err = c.renewSecretID(ctx, c.apiClient, conf)
	if err != nil {
		return fmt.Errorf("error replacing secret-id: %w", err)
	}
	return nil
}

func (c *VaultClientController) renewToken(ctx context.Context) error {
	c.logger.Info("renew vault access token")

	conf, err := getVaultClientConfig(ctx, c.storage)
	if err != nil {
		return err
	}
	err = c.loginByApproleAndUpdateToken(c.apiClient, conf)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	c.logger.Info("vault access token is renewed")
	return nil
}

func (c *VaultClientController) loginByApproleAndUpdateToken(client *api.Client, conf *vaultAccessConfig) error {
	data := map[string]interface{}{
		"role_id":   conf.RoleID,
		"secret_id": conf.SecretID,
	}
	logger := c.logger.Named("Login")
	logger.Debug("started")
	defer logger.Debug("exit")
	loginPath := fmt.Sprintf("%s/login", conf.ApproleMountPoint)
	secret, err := client.Logical().Write(loginPath, data)
	if err != nil {
		return fmt.Errorf("write %s: %w", loginPath, err)
	}
	if secret == nil {
		return fmt.Errorf("write %s: login response is empty", loginPath)
	}
	if secret.Auth == nil {
		return fmt.Errorf("write %s: login response does not contain Auth", loginPath)
	}
	client.SetToken(secret.Auth.ClientToken)
	logger.Debug("normal finish")
	return nil
}

func (c *VaultClientController) renewSecretID(ctx context.Context, apiClient *api.Client,
	conf *vaultAccessConfig) error {
	// login with current secret id if no login current
	newSecretID, err := c.genNewSecretID(apiClient, conf)
	if err != nil {
		return err
	}

	// save new secret id in store
	oldSecretID := conf.SecretID
	conf.SecretID = newSecretID
	conf.LastRenewTime = time.Now()

	err = saveVaultClientConfig(ctx, c.storage, conf)
	if err != nil {
		return err
	}

	err = c.loginByApproleAndUpdateToken(apiClient, conf)
	if err != nil {
		return err
	}

	// delete old secret from vault
	if oldSecretID != "" {
		return c.deleteSecretID(apiClient, conf, oldSecretID)
	}
	return nil
}

func (c *VaultClientController) genNewSecretID(client *api.Client, conf *vaultAccessConfig) (string, error) {
	createSecretIDPath := fmt.Sprintf("%s/role/%s/secret-id", conf.ApproleMountPoint, conf.RoleName)
	secret, err := client.Logical().Write(createSecretIDPath, nil)
	if err != nil {
		return "", fmt.Errorf("write %s: %w", createSecretIDPath, err)
	}
	if secret == nil {
		return "", fmt.Errorf("write %s: write response is empty", createSecretIDPath)
	}
	if secret.Data == nil {
		return "", fmt.Errorf("write %s: write response  does not contain Data", createSecretIDPath)
	}

	secretIDRaw, ok := secret.Data["secret_id"]
	secretID, okCast := secretIDRaw.(string)
	if !ok || !okCast {
		return "", fmt.Errorf("invalid content of data: %v", secret.Data)
	}
	return secretID, nil
}

func (c *VaultClientController) deleteSecretID(client *api.Client, conf *vaultAccessConfig, secretID string) error {
	deleteSecretIDPath := fmt.Sprintf("%s/role/%s/secret-id/destroy", conf.ApproleMountPoint, conf.RoleName)
	_, err := client.Logical().Write(deleteSecretIDPath, map[string]interface{}{
		"secret_id": secretID,
	})
	return err
}
