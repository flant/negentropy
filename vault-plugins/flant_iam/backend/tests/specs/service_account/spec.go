package serviceaccount

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

var _ = Describe("ServiceAccount", func() {
	var tenant model.Tenant
	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
	}, 1.0)
	It("can be created", func() {
		createPayload := fixtures.RandomServiceAccountCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				saData := json.Get("service_account")
				Expect(saData.Map()).To(HaveKey("uuid"))
				Expect(saData.Map()).To(HaveKey("identifier"))
				Expect(saData.Map()).To(HaveKey("full_identifier"))
				Expect(saData.Map()).To(HaveKey("allowed_cidrs"))
				Expect(saData.Map()).To(HaveKey("origin"))
				Expect(saData.Map()).To(HaveKey("token_ttl"))
				Expect(saData.Map()).To(HaveKey("token_max_ttl"))
				Expect(saData.Get("uuid").String()).To(HaveLen(36))
				Expect(saData.Get("resource_version").String()).To(HaveLen(36))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
		createdData := specs.ConvertToGJSON(sa)

		TestAPI.Read(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("service_account"), "extensions")
			},
		}, nil)
	})

	It("can be updated", func() {
		sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
		updatePayload := fixtures.RandomServiceAccountCreatePayload()
		updatePayload["tenant_uuid"] = sa.TenantUUID
		updatePayload["resource_version"] = sa.Version

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
			"expectPayload": func(json gjson.Result) {
				updateData = json
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json, "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)

		TestAPI.Delete(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
		}, nil)

		deletedData := TestAPI.Read(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
			"expectStatus":    api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("service_account.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)

		TestAPI.List(api.Params{
			"tenant": sa.TenantUUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("service_accounts").Array(),
					specs.ConvertToGJSON(sa), "extensions")
			},
		}, url.Values{})
	})

	// It("can not be created with privileged", func() {

	Context("after deletion", func() {
		It("can't be deleted", func() {
			sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
			TestAPI.Delete(api.Params{
				"tenant":          sa.TenantUUID,
				"service_account": sa.UUID,
			}, nil)

			TestAPI.Delete(api.Params{
				"tenant":          sa.TenantUUID,
				"service_account": sa.UUID,
				"expectStatus":    api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			sa := specs.CreateRandomServiceAccount(TestAPI, tenant.UUID)
			TestAPI.Delete(api.Params{
				"tenant":          sa.TenantUUID,
				"service_account": sa.UUID,
			}, nil)

			updatePayload := fixtures.RandomServiceAccountCreatePayload()
			updatePayload["tenant_uuid"] = sa.TenantUUID
			updatePayload["resource_version"] = sa.Version
			TestAPI.Update(api.Params{
				"tenant":          sa.TenantUUID,
				"service_account": sa.UUID,
				"expectStatus":    api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})
