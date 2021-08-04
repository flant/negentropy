package usermultipass

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

var (
	TestAPI   api.TestAPI
	TenantAPI api.TestAPI
	UserAPI   api.TestAPI
	ConfigAPI api.ConfigAPI
)

var _ = Describe("User Multipass", func() {
	var (
		tenant model.Tenant
		user   model.User
	)

	BeforeSuite(func() {
		ConfigAPI.GenerateCSR()

		ConfigAPI.ConfigureKafka("cert", []string{"192.168.1.1:9093"})

		ConfigAPI.EnableJWT()

		tenant = specs.CreateRandomTenant(TenantAPI)
		user = specs.CreateRandomUser(UserAPI, tenant.UUID)
	}, 1.0)

	It("can be created", func() {
		createPayload := fixtures.RandomUserMultipassCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID
		createPayload["owner_uuid"] = user.UUID

		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				multipassData := json.Get("multipass")
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
				Expect(multipassData.Get("uuid").String()).ToNot(HaveLen(10))
			},
			"tenant": tenant.UUID,
			"user":   user.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		multipass := specs.CreateRandomUserMultipass(TestAPI, tenant.UUID, user.UUID)
		createdData := specs.ConvertToGJSON(multipass)

		TestAPI.Read(api.Params{
			"tenant":    multipass.TenantUUID,
			"user":      multipass.OwnerUUID,
			"multipass": multipass.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("multipass"), "extensions")
			},
		}, nil)
	})

	It("can be deleted", func() {
		multipass := specs.CreateRandomUserMultipass(TestAPI, tenant.UUID, user.UUID)

		TestAPI.Delete(api.Params{
			"tenant":    multipass.TenantUUID,
			"user":      multipass.OwnerUUID,
			"multipass": multipass.UUID,
		}, nil)

		deletedData := TestAPI.Read(api.Params{
			"tenant":       multipass.TenantUUID,
			"user":         multipass.OwnerUUID,
			"multipass":    multipass.UUID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedData.Get("multipass.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		multipass := specs.CreateRandomUserMultipass(TestAPI, tenant.UUID, user.UUID)

		TestAPI.List(api.Params{
			"tenant": multipass.TenantUUID,
			"user":   multipass.OwnerUUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("multipasses").Array(),
					specs.ConvertToGJSON(multipass), "extensions")
			},
		}, url.Values{})
	})
})
