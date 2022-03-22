package serviceaccountpassword

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
)

var (
	TestAPI           api.TestAPI
	ServiceAccountAPI api.TestAPI
	TenantAPI         api.TestAPI
)

var _ = Describe("ServiceAccountPassword", func() {
	var tenant model.Tenant
	var serviceAccount model.ServiceAccount
	var serviceAccountPasswordID model.ServiceAccountPasswordUUID
	var createdSAP gjson.Result
	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
		serviceAccount = specs.CreateRandomServiceAccount(ServiceAccountAPI, tenant.UUID)
	}, 1.0)

	It("can be created", func() {
		createPayload := map[string]interface{}{
			"description":   "test_password",
			"allowed_cidrs": "127.0.0.1,127.0.0.2,127.0.0.3",
			"allowed_roles": "r1,r2",
			"ttl":           100,
		}

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				Expect(json.Map()).To(HaveKey("password"))
				sapData := json.Get("password")
				Expect(sapData.Map()).To(HaveKey("uuid"))
				Expect(sapData.Get("uuid").String()).To(HaveLen(36))
				serviceAccountPasswordID = sapData.Get("uuid").String()
				Expect(sapData.Map()).To(HaveKey("tenant_uuid"))
				Expect(sapData.Get("tenant_uuid").String()).To(Equal(tenant.UUID))
				Expect(sapData.Map()).To(HaveKey("archiving_timestamp"))
				Expect(sapData.Get("archiving_timestamp").Int()).To(Equal(int64(0)))
				Expect(sapData.Map()).To(HaveKey("archiving_hash"))
				Expect(sapData.Get("archiving_hash").Int()).To(Equal(int64(0)))
				Expect(sapData.Map()).To(HaveKey("owner_uuid"))
				Expect(sapData.Get("owner_uuid").String()).To(Equal(serviceAccount.UUID))
				Expect(sapData.Map()).To(HaveKey("description"))
				Expect(sapData.Get("description").String()).To(Equal("test_password"))
				Expect(sapData.Map()).To(HaveKey("allowed_cidrs"))
				Expect(sapData.Get("allowed_cidrs").Array()).To(HaveLen(3))
				Expect(sapData.Map()).To(HaveKey("allowed_roles"))
				Expect(sapData.Get("allowed_roles").Array()).To(HaveLen(2))
				Expect(sapData.Map()).To(HaveKey("allowed_roles"))
				Expect(sapData.Get("allowed_roles").Array()).To(HaveLen(2))
				Expect(sapData.Map()).To(HaveKey("ttl"))
				Expect(sapData.Get("ttl").Int()).To(Equal(int64(100000000000)))
				Expect(sapData.Map()).To(HaveKey("valid_till"))
				Expect(sapData.Map()).To(HaveKey("secret"))
				createdSAP = sapData
			},
			"tenant":          tenant.UUID,
			"service_account": serviceAccount.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		TestAPI.Read(api.Params{
			"tenant":          tenant.UUID,
			"service_account": serviceAccount.UUID,
			"password":        serviceAccountPasswordID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdSAP, json.Get("password"), "secret")
			},
		}, nil)
	})

	It("can be listed", func() {
		TestAPI.List(api.Params{
			"tenant":          tenant.UUID,
			"service_account": serviceAccount.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("passwords").Array(),
					createdSAP, "secret")
			},
		}, url.Values{})
	})

	// It("can't be updated", func() {})

	It("can be deleted", func() {
		TestAPI.Delete(api.Params{
			"tenant":          tenant.UUID,
			"service_account": serviceAccount.UUID,
			"password":        serviceAccountPasswordID,
		}, nil)

		deletedData := TestAPI.Read(api.Params{
			"tenant":          tenant.UUID,
			"service_account": serviceAccount.UUID,
			"password":        serviceAccountPasswordID,
			"expectStatus":    api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("password.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	// It("can not be created with privileged", func() {})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			TestAPI.Delete(api.Params{
				"tenant":          tenant.UUID,
				"service_account": serviceAccount.UUID,
				"password":        serviceAccountPasswordID,
				"expectStatus":    api.ExpectExactStatus(400),
			}, nil)
		})
	})
})
