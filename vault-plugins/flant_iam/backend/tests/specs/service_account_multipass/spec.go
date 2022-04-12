package serviceaccountmultipass

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	cfg_api "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
)

var (
	TestAPI           api.TestAPI
	TenantAPI         api.TestAPI
	ServiceAccountAPI api.TestAPI
	ConfigAPI         cfg_api.ConfigAPI
)

var _ = Describe("ServiceAccount Multipass", func() {
	var (
		tenant         model.Tenant
		serviceAccount model.ServiceAccount
		multipassData  gjson.Result
		multipassID    model.MultipassUUID
	)

	BeforeSuite(func() {
		ConfigAPI.ConfigureKafka("cert", []string{"192.168.1.1:9093"})

		ConfigAPI.EnableJWT()

		tenant = specs.CreateRandomTenant(TenantAPI)
		serviceAccount = specs.CreateRandomServiceAccount(ServiceAccountAPI, tenant.UUID)
	}, 1.0)

	It("can be created", func() {
		createPayload := fixtures.RandomUserMultipassCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID
		createPayload["owner_uuid"] = serviceAccount.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				multipassData = json.Get("multipass")
				Expect(multipassData.Map()).To(HaveKey("uuid"))
				Expect(multipassData.Map()).To(HaveKey("tenant_uuid"))
				Expect(multipassData.Map()).To(HaveKey("owner_uuid"))
				Expect(multipassData.Map()).To(HaveKey("owner_type"))
				Expect(multipassData.Map()).To(HaveKey("description"))
				Expect(multipassData.Map()).To(HaveKey("ttl"))
				Expect(multipassData.Map()).To(HaveKey("max_ttl"))
				Expect(multipassData.Map()).To(HaveKey("allowed_cidrs"))
				Expect(multipassData.Map()).To(HaveKey("allowed_roles"))
				Expect(multipassData.Map()).To(HaveKey("valid_till"))
				Expect(multipassData.Map()).To(HaveKey("archiving_timestamp"))
				Expect(multipassData.Map()).To(HaveKey("archiving_hash"))
				Expect(multipassData.Get("uuid").String()).To(HaveLen(36))
				multipassID = multipassData.Get("uuid").String()
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
			"multipass":       multipassID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(multipassData, json.Get("multipass"))
			},
		}, nil)
	})

	It("can be listed", func() {
		TestAPI.List(api.Params{
			"tenant":          tenant.UUID,
			"service_account": serviceAccount.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("multipasses").Array(),
					multipassData)
			},
		}, url.Values{})
	})

	It("can be deleted", func() {
		TestAPI.Delete(api.Params{
			"tenant":          tenant.UUID,
			"service_account": serviceAccount.UUID,
			"multipass":       multipassID,
		}, nil)

		deletedData := TestAPI.Read(api.Params{
			"tenant":          tenant.UUID,
			"service_account": serviceAccount.UUID,
			"multipass":       multipassID,
			"expectStatus":    api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("multipass.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			TestAPI.Delete(api.Params{
				"tenant":          tenant.UUID,
				"service_account": serviceAccount.UUID,
				"multipass":       multipassID,
				"expectStatus":    api.ExpectExactStatus(400),
			}, nil)
		})
	})
})
