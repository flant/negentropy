package group

import (
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
	TestAPI   api.TestAPI
	TenantAPI api.TestAPI
	UserAPI   api.TestAPI
)

var _ = Describe("Group", func() {
	var (
		tenant model.Tenant
		user   model.User
	)

	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
		user = specs.CreateRandomUser(UserAPI, tenant.UUID)
	}, 1.0)

	It("can be created", func() {
		createPayload := fixtures.RandomGroupCreatePayload()
		delete(createPayload, "tenant_uuid")
		createPayload["members"] = map[string]interface{}{
			"type": "user",
			"uuid": user.UUID,
		}

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				userData := json.Get("group")
				Expect(userData.Map()).To(HaveKey("uuid"))
				Expect(userData.Map()).To(HaveKey("tenant_uuid"))
				Expect(userData.Map()).To(HaveKey("resource_version"))
				Expect(userData.Map()).To(HaveKey("identifier"))
				Expect(userData.Map()).To(HaveKey("full_identifier"))
				Expect(userData.Map()).To(HaveKey("members"))
				Expect(userData.Map()).To(HaveKey("archiving_timestamp"))
				Expect(userData.Map()).To(HaveKey("archiving_hash"))
				Expect(userData.Get("uuid").String()).To(HaveLen(36))
				Expect(userData.Get("resource_version").String()).To(HaveLen(36))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		group := specs.CreateRandomGroupWithUser(TestAPI, tenant.UUID, user.UUID)
		createdData := specs.ConvertToGJSON(group)

		TestAPI.Read(api.Params{
			"tenant": group.TenantUUID,
			"group":  group.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("group"), "extensions")
			},
		}, nil)
	})

	It("can be updated", func() {
		group := specs.CreateRandomGroupWithUser(TestAPI, tenant.UUID, user.UUID)
		updatePayload := fixtures.RandomGroupCreatePayload()
		updatePayload["tenant_uuid"] = group.TenantUUID
		updatePayload["resource_version"] = group.Version
		updatePayload["members"] = map[string]interface{}{
			"type": "user",
			"uuid": user.UUID,
		}

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"tenant": group.TenantUUID,
			"group":  group.UUID,
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant": group.TenantUUID,
			"group":  group.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json, "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		group := specs.CreateRandomGroupWithUser(TestAPI, tenant.UUID, user.UUID)

		TestAPI.Delete(api.Params{
			"tenant": group.TenantUUID,
			"group":  group.UUID,
		}, nil)

		deletedData := TestAPI.Read(api.Params{
			"tenant":       group.TenantUUID,
			"group":        group.UUID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("group.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		group := specs.CreateRandomGroupWithUser(TestAPI, tenant.UUID, user.UUID)

		TestAPI.List(api.Params{
			"tenant": user.TenantUUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("groups").Array(),
					specs.ConvertToGJSON(group), "extensions")
			},
		}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomGroupCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID
		createPayload["members"] = map[string]interface{}{
			"type": "user",
			"uuid": user.UUID,
		}
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				groupData := json.Get("group")
				Expect(groupData.Map()).To(HaveKey("uuid"))
				Expect(groupData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			gr := createGroup(TenantAPI, UserAPI, TestAPI)
			TestAPI.Delete(api.Params{
				"tenant": gr.TenantUUID,
				"group":  gr.UUID,
			}, nil)

			TestAPI.Delete(api.Params{
				"tenant":       gr.TenantUUID,
				"group":        gr.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			gr := createGroup(TenantAPI, UserAPI, TestAPI)
			TestAPI.Delete(api.Params{
				"tenant": gr.TenantUUID,
				"group":  gr.UUID,
			}, nil)

			updatePayload := fixtures.RandomGroupCreatePayload()
			updatePayload["uuid"] = gr.UUID
			updatePayload["tenant_uuid"] = gr.TenantUUID
			updatePayload["resource_version"] = gr.Version
			TestAPI.Update(api.Params{
				"tenant":       gr.TenantUUID,
				"group":        gr.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func createGroup(tenantAPI, userAPI, groupAPI api.TestAPI) *model.Group {
	tenant := specs.CreateRandomTenant(tenantAPI)
	user := specs.CreateRandomUser(userAPI, tenant.UUID)
	group := specs.CreateRandomGroupWithUser(groupAPI, tenant.UUID, user.UUID)
	return &group
}
