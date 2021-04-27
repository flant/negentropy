package vault_client

import (
	"fmt"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
)

type accessClient struct {
	logger      hclog.Logger
	apiClient   *api.Client
	conf        *vaultAccessConfig
	mountPrefix string
	rolePrefix  string
}

func newAccessClient(apiClient *api.Client, accessConf *vaultAccessConfig, logger hclog.Logger) *accessClient {
	rolePrefix := fmt.Sprintf("%s/role/%s", accessConf.ApproleMountPoint, accessConf.RoleName)

	return &accessClient{
		apiClient:   apiClient,
		conf:        accessConf,
		mountPrefix: accessConf.ApproleMountPoint,
		rolePrefix:  rolePrefix,
		logger:      logger,
	}
}

func (c *accessClient) AppRole() *AppRole {
	return &AppRole{
		client: c,
		logger: c.logger,
	}
}

type AppRole struct {
	client *accessClient
	logger hclog.Logger
}

func (c *AppRole) Login() (*api.SecretAuth, error) {
	data := map[string]interface{}{
		"role_id":   c.client.conf.RoleId,
		"secret_id": c.client.conf.SecretId,
	}
	secret, err := c.client.apiClient.Logical().Write(c.pluginPath("/login"), data)

	if err != nil {
		c.logger.Error(fmt.Sprintf("error while login %v", err))
		return nil, err
	}

	if secret.Auth == nil {
		return nil, fmt.Errorf("login error does not contain Auth")
	}

	return secret.Auth, nil
}

func (c *AppRole) GenNewSecretId() (string, error) {
	secret, err := c.client.apiClient.Logical().Write(c.rolePath("/secret-id"), nil)

	if err != nil {
		return "", err
	}

	if secret.Data == nil {
		return "", fmt.Errorf("login error does not contain Auth")
	}

	secretIdRaw, ok := secret.Data["secret_id"]
	secretId, okCast := secretIdRaw.(string)
	if !ok || !okCast {
		return "", fmt.Errorf("login error does not Auth.secret_id")
	}

	return secretId, nil
}

func (c *AppRole) DeleteSecretId(secretId string) error {
	_, err := c.client.apiClient.Logical().Write(c.rolePath("/secret-id/destroy"), map[string]interface{}{
		"secret_id": secretId,
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *AppRole) pluginPath(p string) string {
	return fmt.Sprintf("%s%s", c.client.mountPrefix, p)
}

func (c *AppRole) rolePath(p string) string {
	return fmt.Sprintf("%s%s", c.client.rolePrefix, p)
}
