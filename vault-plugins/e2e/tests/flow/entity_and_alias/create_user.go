package entity_and_alias

import (
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/auth_source"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/configure"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tenant"
	user "github.com/flant/negentropy/vault-plugins/e2e/tests/lib/user"
	flant_vault_api "github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/hashicorp/vault/api"
	"time"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Creating entity and entity aliases", func() {
	token := lib.GetSecondRootToken()
	iamAuthClient := configure.GetClient(token)
	iamAuthClient.SetToken(token)

	iamClient := configure.GetClient(token)
	iamClient.SetToken(token)


	identityApi := flant_vault_api.NewIdentityAPI(configure.GetClient(token))

	It("with auth sources", func() {
		role := configure.CreateGoodRole(token)
		configure.ConfigureVaultAccess(token, lib.IamAuthPluginPath, role)
		sources := auth_source.GenerateSources()
		for _, s := range sources {
			p, name := s.ToPayload()
			_, err := iamAuthClient.Logical().Write(lib.IamAuthPluginPath + "/auth_source/" + name, p)
			Expect(err).ToNot(HaveOccurred())
		}


		tenant, err := iamClient.Logical().Write(lib.IamPluginPath + "/tenant", tools.ToMap(tenant.GetPayload()))
		Expect(err).ToNot(HaveOccurred())
		tenantUUID := uuidFromResp(tenant, "tenant", "uuid")

		userUUIDResp, err := iamClient.Logical().Write(lib.IamPluginPath + "/tenant/" + tenantUUID + "/user/", tools.ToMap(user.GetPayload()))
		Expect(err).ToNot(HaveOccurred())
		userUUID := uuidFromResp(userUUIDResp, "user", "uuid")

		userRaw, err := iamClient.Logical().Read(lib.IamPluginPath + "/tenant/" + tenantUUID + "/user/" + userUUID)
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(5 * time.Second)

		entity, err := identityApi.EntityApi().GetByName(uuidFromResp(userRaw, "user", "full_identifier"))
		Expect(err).ToNot(HaveOccurred())
		Expect(entity).ToNot(BeNil())

		// for _, s := range sources {
		//	eaName := s.ExpectedEaName(&user)
		//	if eaName != "" {
		//		aliasId, err := identityApi.AliasApi().FindAliasIDByName(eaName, "flant_iam_auth")
		//		Expect(err).ToNot(HaveOccurred())
		//		Expect(aliasId).ToNot(BeEmpty())
		//	}
		// }

	})

})

func uuidFromResp(resp *api.Secret, entityKey, key string) string{
	return resp.Data[entityKey].(map[string]interface{})[key].(string)
}
