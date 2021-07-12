package configure

import (
	"fmt"
	"github.com/hashicorp/vault/api"
	. "github.com/onsi/gomega"
	"net/http"
)

const goodPolicy = `
path "*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
`

const goodPolicyName = "good"
const goodRoleName = goodPolicyName

type GoodAppRole struct {
	Name string
	SecretId string
	ID string
}

func CreateGoodRole(client *http.Client) *GoodAppRole{
	cl, err := api.NewClient(&api.Config{
		HttpClient: client,
	})
	Expect(err).To(BeNil())

	policyFromServer, err := cl.Sys().GetPolicy(goodPolicyName)
	Expect(err).To(BeNil())

	if policyFromServer == "" {
		err := cl.Sys().PutPolicy(goodPolicyName, goodPolicy)
		Expect(err).To(BeNil())
	}

	appRolePath := fmt.Sprintf("/auth/approle/role/%s", goodRoleName)
	roleFromServer, err := cl.Logical().Read(appRolePath)
	Expect(err).To(BeNil())
	if roleFromServer == nil {
		res, err := cl.Logical().Write(appRolePath, map[string]interface{}{
			"secret_id_ttl": "30m",
			"token_ttl": "25m",
			"token_policies": goodPolicyName,
		})
		Expect(err).To(BeNil())
		Expect(res).NotTo(BeNil())
	}

	secretIdData, err := cl.Logical().Read(appRolePath + "/secret-id/lookup")
	Expect(err).To(BeNil())

	if secretIdData == nil {
		secretIdData, err = cl.Logical().Write(appRolePath + "/secret-id", nil)
		Expect(err).To(BeNil())
	}

	roleIdData, err := cl.Logical().Read(appRolePath + "/secret-id/lookup")
	Expect(err).To(BeNil())
	Expect(roleIdData).ToNot(BeNil())

	return &GoodAppRole{
		Name: goodRoleName,
		SecretId: secretIdData.Data["secret_id"].(string),
		ID: roleFromServer.Data["role_id"].(string),
	}
}

func ConfigureVaultAccess(client *http.Client, modulePath string, appRole *GoodAppRole) {
	cl, err := api.NewClient(&api.Config{
		HttpClient: client,
	})
	Expect(err).To(BeNil())

	_, err = cl.Logical().Write(modulePath + "/configure_vault_access", map[string]interface{}{
		"vault_api_url": "http://127.0.0.1:8200",
		"vault_api_host": "vault_host",
		"role_name": appRole.Name,
		"secret_id_ttl": "30m",
		"approle_mount_point": "/auth/approle/",
		"secret_id": appRole.SecretId,
		"role_id": appRole.ID,
		"vault_api_ca": "",
	})

	Expect(err).To(BeNil())
}
