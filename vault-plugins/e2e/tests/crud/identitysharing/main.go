package tenant

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tenant"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

var _ = Describe("Identity sharing", func() {
	rootClient := lib.NewConfiguredIamVaultClient()
	tenantsAPI := lib.NewTenantAPI(rootClient)
	identitySharingAPI := lib.NewIdentitySharingAPI(rootClient)

	var sourceTenantID, targetTenantID string

	BeforeSuite(func() {
		t1 := tenant.GetPayload()
		res := tenantsAPI.Create(nil, url.Values{}, t1)
		sourceTenantID = res.Get("tenant.uuid").String()

		t2 := tenant.GetPayload()
		res = tenantsAPI.Create(nil, url.Values{}, t2)
		targetTenantID = res.Get("tenant.uuid").String()
	})

	var createdData gjson.Result

	It("can be created", func() {
		params := tools.Params{
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				is := data.Get("identity_sharing")

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
		createdData = identitySharingAPI.Create(params, url.Values{}, data)
	})

	It("can be read", func() {
		identitySharingAPI.Read(tools.Params{
			"uuid":        createdData.Get("identity_sharing.uuid").String(),
			"tenant_uuid": sourceTenantID,
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				Expect(createdData).To(Equal(data))
			},
		}, nil)
	})

	It("can be listed", func() {
		list := identitySharingAPI.List(tools.Params{
			"tenant_uuid": sourceTenantID,
		}, url.Values{})
		Expect(list.Get("identity_sharings").Array()).To(HaveLen(1))
		Expect(list.Get("identity_sharings").Array()[0].Get("uuid").String()).To(BeEquivalentTo(createdData.Get("identity_sharing.uuid").String()))
	})

	It("can be deleted", func() {
		identitySharingAPI.Delete(tools.Params{
			"uuid":        createdData.Get("identity_sharing.uuid").String(),
			"tenant_uuid": sourceTenantID,
		}, nil)

		identitySharingAPI.Read(tools.Params{
			"uuid":         createdData.Get("identity_sharing.uuid").String(),
			"tenant_uuid":  sourceTenantID,
			"expectStatus": tools.ExpectExactStatus(404),
		}, nil)
	})

})
