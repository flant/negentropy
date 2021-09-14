package configure

import (
	"fmt"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/gomega"
)

const goodPolicy = `
path "*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
`

const (
	goodPolicyName = "good"
	goodRoleName   = goodPolicyName
)

type GoodAppRole struct {
	Name     string
	SecretId string
	ID       string
}

// GetClientWithToken create vault client with customized token and addr
// example: GetClientWithToken("token","http://127.0.0.1:8200")
func GetClientWithToken(token string, addr string) *api.Client {
	cl, err := api.NewClient(api.DefaultConfig())
	Expect(err).To(BeNil())

	cl.SetToken(token)
	err = cl.SetAddress(addr)
	Expect(err).To(BeNil())

	return cl
}

func CreatePolicy(vaultClient *api.Client, name, content string) {
	policyFromServer, err := vaultClient.Sys().GetPolicy(name)
	Expect(err).To(BeNil())

	if policyFromServer == "" {
		err := vaultClient.Sys().PutPolicy(name, content)
		Expect(err).To(BeNil())
	}
}

func CreateGoodRole(vaultClient *api.Client) *GoodAppRole {
	CreatePolicy(vaultClient, goodPolicyName, goodPolicy)

	EnableAuthPlugin("flant_iam_auth", "flant_iam_auth", vaultClient)
	EnableAuthPlugin("approle", "approle", vaultClient)

	appRolePath := fmt.Sprintf("/auth/approle/role/%s", goodRoleName)
	roleFromServer, err := vaultClient.Logical().Read(appRolePath)
	Expect(err).To(BeNil())
	if roleFromServer == nil {
		res, err := vaultClient.Logical().Write(appRolePath, map[string]interface{}{
			"secret_id_ttl":  "30m",
			"token_ttl":      "25m",
			"token_policies": []string{goodPolicyName},
		})
		Expect(err).To(BeNil())
		Expect(res).To(BeNil())
	}

	secretIdData, err := vaultClient.Logical().Write(appRolePath+"/secret-id", nil)
	Expect(err).To(BeNil())

	roleIdData, err := vaultClient.Logical().Read(appRolePath + "/role-id")
	Expect(err).To(BeNil())
	Expect(roleIdData).ToNot(BeNil())

	return &GoodAppRole{
		Name:     goodRoleName,
		SecretId: secretIdData.Data["secret_id"].(string),
		ID:       roleIdData.Data["role_id"].(string),
	}
}

func ConfigureVaultAccess(vaultClient *api.Client, pluginPath string, appRole *GoodAppRole) {
	_, err := vaultClient.Logical().Write(pluginPath+"/configure_vault_access", map[string]interface{}{
		"vault_addr":            "http://127.0.0.1:8200",
		"vault_tls_server_name": "vault_host",
		"role_name":             appRole.Name,
		"secret_id_ttl":         "30m",
		"approle_mount_point":   "/auth/approle/",
		"secret_id":             appRole.SecretId,
		"role_id":               appRole.ID,
		"vault_api_ca":          "",
	})

	Expect(err).To(BeNil())
}

func EnableAuthPlugin(plugin, path string, vaultClient *api.Client) {
	err := vaultClient.Sys().EnableAuthWithOptions(path, &api.EnableAuthOptions{
		Type: plugin,
	})
	if err != nil {
		Expect(err).ToNot(MatchError("path is already in use at"))
	}
}
