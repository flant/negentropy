package vault_client

import (
	"fmt"
	"github.com/hashicorp/vault/api"
	"strings"
)

type accessClient struct {
	apiClient   *api.Client
	conf        *vaultAccessConfig
	mountPrefix string
	rolePrefix  string
}

func newAccessClient(apiClient *api.Client, accessConf *vaultAccessConfig) *accessClient {
	apiVer := "/v1"
	if !strings.HasPrefix(accessConf.ApproleMountPoint, "/") {
		apiVer = fmt.Sprintf("%s/", apiVer)
	}

	mountPrefix := fmt.Sprintf("%s%s", apiVer, accessConf.ApproleMountPoint)
	rolePrefix := fmt.Sprintf("%s%s", mountPrefix, accessConf.RoleName)

	return &accessClient{
		apiClient:   apiClient,
		conf:        accessConf,
		mountPrefix: mountPrefix,
		rolePrefix:  rolePrefix,
	}
}

func (c *accessClient) AppRole() *AppRole {
	return &AppRole{client: c}
}

type AppRole struct {
	client *accessClient
}

func (c *AppRole) Login() (*api.SecretAuth, error) {
	secret, err := c.client.apiClient.Logical().Write(c.pluginPath("/login"), map[string]interface{}{
		"role_id":   c.client.conf.SecretId,
		"secret_id": c.client.conf.RoleId,
	})

	if err != nil {
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
	return fmt.Sprintf("%s/%s", c.client.mountPrefix, p)
}

func (c *AppRole) rolePath(p string) string {
	return fmt.Sprintf("%s%s", c.client.rolePrefix, p)
}
