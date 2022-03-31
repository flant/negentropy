package user

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
	TestAPI   api.TestAPI
	TenantAPI api.TestAPI
)

var _ = Describe("User", func() {
	var tenant model.Tenant
	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomUserAtTenantWithIdentifier(tenant.UUID, identifier, statusCodeCondition)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

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
				Expect(userData.Get("origin").String()).To(Equal("iam"))
				Expect(userData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(userData.Get("resource_version").String()).ToNot(HaveLen(10))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	Context("tenant uniqueness of user identifier", func() {
		identifier := uuid.New()
		It("Can be created user with some identifier", func() {
			tryCreateRandomUserAtTenantWithIdentifier(tenant.UUID, identifier, "%d == 201")
		})
		It("Can not be the same identifier at the same tenant", func() {
			tryCreateRandomUserAtTenantWithIdentifier(tenant.UUID, identifier, "%d >= 400")
		})
		It("Can be same identifier at other tenant", func() {
			tenant = specs.CreateRandomTenant(TenantAPI)
			tryCreateRandomUserAtTenantWithIdentifier(tenant.UUID, identifier, "%d == 201")
		})
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
				userData := updateData.Get("user")
				Expect(userData.Map()).To(HaveKey("origin"))
				Expect(userData.Get("origin").String()).To(Equal("iam"))
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

func tryCreateRandomUserAtTenantWithIdentifier(tenantUUID string,
	userIdentifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomUserCreatePayload()
	payload["identifier"] = userIdentifier

	params := api.Params{
		"tenant":       tenantUUID,
		"expectStatus": api.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}
