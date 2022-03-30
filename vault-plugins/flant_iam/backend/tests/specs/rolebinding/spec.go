package rolebinding

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TenantAPI         api.TestAPI
	UserAPI           api.TestAPI
	ServiceAccountAPI api.TestAPI
	GroupAPI          api.TestAPI
	RoleAPI           api.TestAPI
	TestAPI           api.TestAPI
)

var _ = Describe("Role binding", func() {
	var (
		tenant model.Tenant
		user   model.User
		sa     model.ServiceAccount
		group  model.Group
	)

	BeforeSuite(func() {
		specs.CreateRoles(RoleAPI, fixtures.Roles()...)
		tenant = specs.CreateRandomTenant(TenantAPI)
		user = specs.CreateRandomUser(UserAPI, tenant.UUID)
		sa = specs.CreateRandomServiceAccount(ServiceAccountAPI, tenant.UUID)
		group = specs.CreateRandomGroupWithUser(GroupAPI, tenant.UUID, user.UUID)
	})

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				payload := fixtures.RandomRoleBindingCreatePayload()
				payload["members"] = []map[string]interface{}{
					{"type": model.UserType, "uuid": user.UUID},
				}
				payload["identifier"] = identifier

				params := api.Params{
					"tenant":       tenant.UUID,
					"expectStatus": api.ExpectStatus(statusCodeCondition),
				}

				TestAPI.Create(params, nil, payload)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score", "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomRoleBindingCreatePayload()
		delete(createPayload, "resource_version")
		delete(createPayload, "uuid")
		delete(createPayload, "archiving_timestamp")
		delete(createPayload, "archiving_hash")
		delete(createPayload, "tenant_uuid")
		createPayload["members"] = []map[string]interface{}{
			{"type": model.UserType, "uuid": user.UUID},
			{"type": model.ServiceAccountType, "uuid": sa.UUID},
			{"type": model.GroupType, "uuid": group.UUID},
		}

		params := api.Params{
			"expectStatus": api.ExpectExactStatus(201),
			"expectPayload": func(json gjson.Result) {
				rbData := json.Get("role_binding")
				Expect(rbData.Map()).To(HaveKey("valid_till"))
				Expect(rbData.Map()).To(HaveKey("require_mfa"))
				Expect(rbData.Map()).To(HaveKey("groups"))
				Expect(rbData.Map()).To(HaveKey("archiving_timestamp"))
				Expect(rbData.Map()).To(HaveKey("users"))
				Expect(rbData.Map()).To(HaveKey("members"))
				Expect(rbData.Map()).To(HaveKey("tenant_uuid"))
				Expect(rbData.Map()).To(HaveKey("resource_version"))
				Expect(rbData.Map()).To(HaveKey("service_accounts"))
				Expect(rbData.Map()).To(HaveKey("any_project"))
				Expect(rbData.Map()).To(HaveKey("archiving_hash"))
				Expect(rbData.Map()).To(HaveKey("identifier"))
				Expect(rbData.Map()).To(HaveKey("projects"))
				Expect(rbData.Map()).To(HaveKey("roles"))
				Expect(rbData.Map()).To(HaveKey("origin"))
				Expect(rbData.Get("origin").String()).To(Equal("iam"))
				Expect(rbData.Get("uuid").String()).To(HaveLen(36))
				Expect(rbData.Get("resource_version").String()).To(HaveLen(36))
			},
			"tenant": tenant.UUID,
		}

		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		createdRB := TestAPI.Create(api.Params{"tenant": tenant.UUID}, url.Values{}, fixtures.RandomRoleBindingCreatePayload())

		TestAPI.Read(api.Params{
			"tenant":       tenant.UUID,
			"role_binding": createdRB.Get("role_binding.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				Expect(createdRB.Get("role_binding")).To(Equal(json.Get("role_binding")))
			},
		}, nil)
	})

	It("can be updated", func() {
		createdRB := TestAPI.Create(api.Params{"tenant": tenant.UUID}, url.Values{}, fixtures.RandomRoleBindingCreatePayload())
		updatePayload := fixtures.RandomRoleBindingCreatePayload()
		updatePayload["tenant_uuid"] = createdRB.Get("role_binding.tenant_uuid").String()
		updatePayload["resource_version"] = createdRB.Get("role_binding.resource_version").String()
		updatePayload["members"] = []map[string]interface{}{
			{"type": model.UserType, "uuid": user.UUID},
			{"type": model.ServiceAccountType, "uuid": sa.UUID},
			{"type": model.GroupType, "uuid": group.UUID},
		}

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"tenant":       createdRB.Get("role_binding.tenant_uuid").String(),
			"role_binding": createdRB.Get("role_binding.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant":       createdRB.Get("role_binding.tenant_uuid").String(),
			"role_binding": createdRB.Get("role_binding.uuid").String(),
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json)
				rbData := json.Get("role_binding")
				Expect(rbData.Map()).To(HaveKey("origin"))
				Expect(rbData.Get("origin").String()).To(Equal("iam"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		createdRB := TestAPI.Create(api.Params{"tenant": tenant.UUID}, url.Values{}, fixtures.RandomRoleBindingCreatePayload())
		TestAPI.Delete(api.Params{
			"tenant":       createdRB.Get("role_binding.tenant_uuid").String(),
			"role_binding": createdRB.Get("role_binding.uuid").String(),
		}, nil)

		deletedRBData := TestAPI.Read(api.Params{
			"tenant":       tenant.UUID,
			"role_binding": createdRB.Get("role_binding.uuid").String(),

			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedRBData.Get("role_binding.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		createdRB := TestAPI.Create(api.Params{"tenant": tenant.UUID}, url.Values{}, fixtures.RandomRoleBindingCreatePayload())

		TestAPI.List(api.Params{
			"tenant": tenant.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("role_bindings").Array(),
					createdRB.Get("role_binding"))
			},
		}, url.Values{})
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			createdRB := TestAPI.Create(api.Params{"tenant": tenant.UUID}, url.Values{}, fixtures.RandomRoleBindingCreatePayload())
			TestAPI.Delete(api.Params{
				"tenant":       createdRB.Get("role_binding.tenant_uuid").String(),
				"role_binding": createdRB.Get("role_binding.uuid").String(),
			}, nil)

			TestAPI.Delete(api.Params{
				"tenant":       createdRB.Get("role_binding.tenant_uuid").String(),
				"role_binding": createdRB.Get("role_binding.uuid").String(),
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			createdRB := TestAPI.Create(api.Params{"tenant": tenant.UUID}, url.Values{}, fixtures.RandomRoleBindingCreatePayload())
			TestAPI.Delete(api.Params{
				"tenant":       createdRB.Get("role_binding.tenant_uuid").String(),
				"role_binding": createdRB.Get("role_binding.uuid").String(),
			}, nil)

			updatePayload := fixtures.RandomRoleBindingCreatePayload()
			updatePayload["tenant_uuid"] = createdRB.Get("role_binding.tenant_uuid").String()
			updatePayload["resource_version"] = createdRB.Get("role_binding.resource_version").String()
			updatePayload["members"] = []map[string]interface{}{
				{"type": model.UserType, "uuid": user.UUID},
				{"type": model.ServiceAccountType, "uuid": sa.UUID},
				{"type": model.GroupType, "uuid": group.UUID},
			}
			TestAPI.Update(api.Params{
				"tenant":       createdRB.Get("role_binding.tenant_uuid").String(),
				"role_binding": createdRB.Get("role_binding.uuid").String(),
				"expectStatus": api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})
