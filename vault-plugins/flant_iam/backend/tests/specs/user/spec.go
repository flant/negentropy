package user

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
)

var (
	TestAPI   api.TestAPI
	TenantAPI api.TestAPI
)

var _ = Describe("User", func() {
	var tenant model.Tenant
	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
	}, 1.0)
	It("can be created", func() {
		createPayload := fixtures.RandomUserCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				userData := json.Get("user")
				Expect(userData.Map()).To(HaveKey("uuid"))
				Expect(userData.Map()).To(HaveKey("identifier"))
				Expect(userData.Map()).To(HaveKey("full_identifier"))
				Expect(userData.Map()).To(HaveKey("email"))
				Expect(userData.Map()).To(HaveKey("origin"))
				Expect(userData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(userData.Get("resource_version").String()).ToNot(HaveLen(10))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		user := specs.CreateRandomUser(TestAPI, tenant.UUID)
		createdData := specs.ConvertToGJSON(user)

		TestAPI.Read(api.Params{
			"tenant": user.TenantUUID,
			"user":   user.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("user"), "extensions")
			},
		}, nil)
	})

	It("can be updated", func() {
		user := specs.CreateRandomUser(TestAPI, tenant.UUID)
		updatePayload := fixtures.RandomUserCreatePayload()
		updatePayload["tenant_uuid"] = user.TenantUUID
		updatePayload["resource_version"] = user.Version

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"tenant": user.TenantUUID,
			"user":   user.UUID,
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant": user.TenantUUID,
			"user":   user.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json, "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		user := specs.CreateRandomUser(TestAPI, tenant.UUID)

		TestAPI.Delete(api.Params{
			"tenant": user.TenantUUID,
			"user":   user.UUID,
		}, nil)

		deletedData := TestAPI.Read(api.Params{
			"tenant":       user.TenantUUID,
			"user":         user.UUID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("user.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		user := specs.CreateRandomUser(TestAPI, tenant.UUID)

		TestAPI.List(api.Params{
			"tenant": user.TenantUUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("users").Array(),
					specs.ConvertToGJSON(user), "extensions")
			},
		}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomUserCreatePayload()
		originalUUID := createPayload["uuid"]
		createPayload["tenant_uuid"] = tenant.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				userData := json.Get("user")
				Expect(userData.Map()).To(HaveKey("uuid"))
				Expect(userData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			user := specs.CreateRandomUser(TestAPI, tenant.UUID)
			TestAPI.Delete(api.Params{
				"tenant": user.TenantUUID,
				"user":   user.UUID,
			}, nil)

			TestAPI.Delete(api.Params{
				"tenant":       user.TenantUUID,
				"user":         user.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			user := specs.CreateRandomUser(TestAPI, tenant.UUID)
			TestAPI.Delete(api.Params{
				"tenant": user.TenantUUID,
				"user":   user.UUID,
			}, nil)

			updatePayload := fixtures.RandomUserCreatePayload()
			updatePayload["tenant_uuid"] = user.TenantUUID
			updatePayload["resource_version"] = user.Version
			TestAPI.Update(api.Params{
				"tenant":       user.TenantUUID,
				"user":         user.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})
