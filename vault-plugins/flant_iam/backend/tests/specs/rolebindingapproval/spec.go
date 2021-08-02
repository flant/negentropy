package rolebindingapproval

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

var (
	TestTenantAPI              api.TestAPI
	TestRoleBindingAPI         api.TestAPI
	TestRoleBindingApprovalAPI api.TestAPI
)

var _ = Describe("Role binding approval", func() {
	var tenantID, roleBindingID string
	BeforeSuite(func() {
		res := TestTenantAPI.Create(nil, url.Values{}, fixtures.RandomTenantCreatePayload())
		tenantID = res.Get("tenant.uuid").String()
		res = TestRoleBindingAPI.Create(api.Params{"tenant_uuid": tenantID}, url.Values{}, fixtures.RandomRoleBindingCreatePayload())
		roleBindingID = res.Get("role_binding.uuid").String()
	})

	var createdRB gjson.Result

	It("can be created", func() {
		params := api.Params{
			"expectStatus": api.ExpectExactStatus(200),
			"expectPayload": func(json gjson.Result) {
				ap := json.Get("role_binding_approval")
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

		createdData := TestRoleBindingApprovalAPI.Create(params, url.Values{}, data)
		createdRB = createdData.Get("role_binding_approval")
	})

	It("can be read", func() {
		TestRoleBindingApprovalAPI.Read(api.Params{
			"uuid":              createdRB.Get("uuid").String(),
			"tenant_uuid":       tenantID,
			"role_binding_uuid": roleBindingID,
			"expectPayload": func(json gjson.Result) {
				Expect(createdRB).To(Equal(json.Get("role_binding_approval")))
			},
		}, nil)
	})

	It("can be deleted", func() {
		TestRoleBindingApprovalAPI.Delete(api.Params{
			"uuid":              createdRB.Get("uuid").String(),
			"tenant_uuid":       tenantID,
			"role_binding_uuid": roleBindingID,
		}, nil)

		deletedRBData := TestRoleBindingApprovalAPI.Read(api.Params{
			"uuid":              createdRB.Get("uuid").String(),
			"tenant_uuid":       tenantID,
			"role_binding_uuid": roleBindingID,
			"expectStatus":      api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedRBData.Get("role_binding_approval.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})
})
