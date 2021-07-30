package tenant

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/rolebinding"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tenant"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

var _ = Describe("Role binding approval", func() {
	rootClient := lib.NewConfiguredIamVaultClient()
	tenantsAPI := lib.NewTenantAPI(rootClient)
	roleBindingAPI := lib.NewRoleBindingAPI(rootClient)
	roleBindingApprovalAPI := lib.NewRoleBindingApprovalAPI(rootClient)

	var tenantID, roleBindingID string
	BeforeSuite(func() {
		res := tenantsAPI.Create(nil, url.Values{}, tenant.GetPayload())
		tenantID = res.Get("tenant.uuid").String()
		res = roleBindingAPI.Create(tools.Params{"tenant_uuid": tenantID}, url.Values{}, rolebinding.GetPayload())
		roleBindingID = res.Get("role_binding.uuid").String()
	})

	var createdRB gjson.Result

	It("can be created", func() {
		params := tools.Params{
			"expectStatus": tools.ExpectExactStatus(200),
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				ap := data.Get("role_binding_approval")
				Expect(ap.Get("required_votes").Int()).To(BeEquivalentTo(3))
				Expect(ap.Get("user_uuids").Array()).To(HaveLen(1))
				Expect(ap.Get("group_uuids").Array()).To(HaveLen(1))
				Expect(ap.Get("service_account_uuids").Array()).To(HaveLen(1))
			},
			"tenant_uuid":       tenantID,
			"role_binding_uuid": roleBindingID,
			"uuid":              uuid.New(),
		}

		data := map[string]interface{}{
			"required_votes":   3,
			"users":            []string{uuid.New()},
			"groups":           []string{uuid.New()},
			"service_accounts": []string{uuid.New()},
		}

		createdData := roleBindingApprovalAPI.Create(params, url.Values{}, data)
		createdRB = createdData.Get("role_binding_approval")
	})

	It("can be read", func() {
		roleBindingApprovalAPI.Read(tools.Params{
			"uuid":              createdRB.Get("uuid").String(),
			"tenant_uuid":       tenantID,
			"role_binding_uuid": roleBindingID,
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				Expect(createdRB).To(Equal(data.Get("role_binding_approval")))
			},
		}, nil)
	})

	It("can be deleted", func() {
		roleBindingApprovalAPI.Delete(tools.Params{
			"uuid":              createdRB.Get("uuid").String(),
			"tenant_uuid":       tenantID,
			"role_binding_uuid": roleBindingID,
		}, nil)

		roleBindingApprovalAPI.Read(tools.Params{
			"uuid":              createdRB.Get("uuid").String(),
			"tenant_uuid":       tenantID,
			"role_binding_uuid": roleBindingID,
			"expectStatus":      tools.ExpectExactStatus(404),
		}, nil)
	})
})
