package rolebindingapproval

import (
	"encoding/json"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TenantAPI         api.TestAPI
	UserAPI           api.TestAPI
	ServiceAccountAPI api.TestAPI
	GroupAPI          api.TestAPI
	ProjectAPI        api.TestAPI
	RoleAPI           api.TestAPI
	RoleBindingAPI    api.TestAPI
	TestAPI           api.TestAPI
)

var _ = Describe("Role binding approval", func() {
	var (
		tenant model.Tenant
		user   model.User
		sa     model.ServiceAccount
		group  model.Group
	)

	var roleBindingID string
	BeforeSuite(func() {
		specs.CreateRoles(RoleAPI, fixtures.Roles()...)
		tenant = specs.CreateRandomTenant(TenantAPI)
		user = specs.CreateRandomUser(UserAPI, tenant.UUID)
		sa = specs.CreateServiceAccount(ServiceAccountAPI, tenant.UUID)
		group = specs.CreateRandomGroupWithUser(GroupAPI, tenant.UUID, user.UUID)
		res := RoleBindingAPI.Create(api.Params{"tenant": tenant.UUID}, url.Values{}, fixtures.RandomRoleBindingCreatePayload())
		roleBindingID = res.Get("role_binding.uuid").String()
	})

	var createdRB gjson.Result

	It("can be created", func() {
		approvers := []map[string]interface{}{
			{"type": "user", "uuid": user.UUID},
			{"type": "service_account", "uuid": sa.UUID},
			{"type": "group", "uuid": group.UUID},
		}

		data := map[string]interface{}{
			"required_votes": 3,
			"approvers":      approvers,
		}

		params := api.Params{
			"expectStatus": api.ExpectExactStatus(201),
			"expectPayload": func(js gjson.Result) {
				ap := js.Get("approval")
				Expect(ap.Get("required_votes").Int()).To(BeEquivalentTo(3))
				Expect(ap.Get("approvers").Array()).To(HaveLen(3))
				Expect(ap.Get("approvers").Array()).To(HaveLen(3))

				approversBytes, err := json.Marshal(approvers)
				Expect(err).ToNot(HaveOccurred())

				Expect(ap.Get("approvers").Array()).To(Equal(gjson.ParseBytes(approversBytes).Array()))
				println("Hello!")
			},
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
			"uuid":         uuid.New(),
		}

		createdData := TestAPI.Create(params, url.Values{}, data)
		createdRB = createdData.Get("approval")
	})

	It("can be read", func() {
		TestAPI.Read(api.Params{
			"uuid":         createdRB.Get("uuid").String(),
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
			"expectPayload": func(json gjson.Result) {
				Expect(createdRB).To(Equal(json.Get("approval")))
			},
		}, nil)
	})

	It("can be deleted", func() {
		TestAPI.Delete(api.Params{
			"uuid":         createdRB.Get("uuid").String(),
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
		}, nil)

		deletedRBData := TestAPI.Read(api.Params{
			"uuid":         createdRB.Get("uuid").String(),
			"tenant":       tenant.UUID,
			"role_binding": roleBindingID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedRBData.Get("approval.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})
})
