package identitysharing

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
	TestTenantsAPI         api.TestAPI
	TestIdentitySharingAPI api.TestAPI
)

var _ = Describe("Identity sharing", func() {
	var sourceTenantID, targetTenantID string

	BeforeSuite(func() {
		t1 := fixtures.RandomTenantCreatePayload()
		res := TestTenantsAPI.Create(nil, url.Values{}, t1)
		sourceTenantID = res.Get("tenant.uuid").String()

		t2 := fixtures.RandomTenantCreatePayload()
		res = TestTenantsAPI.Create(nil, url.Values{}, t2)
		targetTenantID = res.Get("tenant.uuid").String()
	})

	var createdData gjson.Result

	It("can be created", func() {
		params := api.Params{
			"expectPayload": func(json gjson.Result) {
				is := json.Get("identity_sharing")

				Expect(is.Map()).To(HaveKey("uuid"))
				Expect(is.Map()).To(HaveKey("source_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("destination_tenant_uuid"))
				Expect(is.Map()).To(HaveKey("groups"))
				Expect(is.Get("groups").Array()).To(HaveLen(3))
			},
			"tenant_uuid": sourceTenantID,
		}
		data := map[string]interface{}{
			"destination_tenant_uuid": targetTenantID,
			"groups":                  []string{uuid.New(), uuid.New(), uuid.New()},
		}
		createdData = TestIdentitySharingAPI.Create(params, url.Values{}, data)
	})

	It("can be read", func() {
		TestIdentitySharingAPI.Read(api.Params{
			"uuid":        createdData.Get("identity_sharing.uuid").String(),
			"tenant_uuid": sourceTenantID,
			"expectPayload": func(json gjson.Result) {
				Expect(createdData).To(Equal(json))
			},
		}, nil)
	})

	It("can be listed", func() {
		list := TestIdentitySharingAPI.List(api.Params{
			"tenant_uuid": sourceTenantID,
		}, url.Values{})
		Expect(list.Get("identity_sharings").Array()).To(HaveLen(1))
		Expect(list.Get("identity_sharings").Array()[0].Get("uuid").String()).To(BeEquivalentTo(createdData.Get("identity_sharing.uuid").String()))
	})

	It("can be deleted", func() {
		TestIdentitySharingAPI.Delete(api.Params{
			"uuid":        createdData.Get("identity_sharing.uuid").String(),
			"tenant_uuid": sourceTenantID,
		}, nil)

		deletedISData := TestIdentitySharingAPI.Read(api.Params{
			"uuid":         createdData.Get("identity_sharing.uuid").String(),
			"tenant_uuid":  sourceTenantID,
			"expectStatus": api.ExpectExactStatus(200),
		}, nil)
		Expect(deletedISData.Get("identity_sharing.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})
})
