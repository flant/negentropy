package tenant

import (
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/featureflag"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tenant"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

var _ = Describe("Tenant feature flags", func() {
	rootClient := lib.GetIamVaultClient(lib.RootToken)
	tenantsAPI := lib.NewTenantAPI(rootClient)
	featureFlagAPI := lib.NewFeatureFlagAPI(rootClient)
	tenantFFApi := lib.NewTenantFeatureFlagAPI(rootClient)

	var tenantID, ffName string
	BeforeSuite(func() {
		res := tenantsAPI.Create(nil, url.Values{}, tenant.GetPayload())
		tenantID = res.Get("tenant.uuid").String()
		res = featureFlagAPI.Create(tools.Params{"tenant_uuid": tenantID}, url.Values{}, featureflag.GetPayload())
		ffName = res.Get("feature_flag.name").String()
	})

	It("can be bound", func() {
		params := tools.Params{
			"expectStatus":      tools.ExpectExactStatus(200),
			"tenant_uuid":       tenantID,
			"feature_flag_name": ffName,
		}

		data := map[string]interface{}{
			"required_votes":   3,
			"users":            []string{uuid.New()},
			"groups":           []string{uuid.New()},
			"service_accounts": []string{uuid.New()},
		}

		_ = tenantFFApi.Create(params, url.Values{}, data)
	})

	It("can be read from tenant", func() {
		tenantsAPI.Read(tools.Params{
			"tenant": tenantID,
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				ffArr := data.Get("tenant.feature_flags").Array()
				Expect(ffArr).To(HaveLen(1))
				Expect(ffArr[0].Get("name").String()).To(Equal(ffName))
				Expect(ffArr[0].Get("enabled_for_new").Bool()).To(BeTrue())
			},
		}, nil)
	})

	It("can be unbound", func() {
		tenantFFApi.Delete(tools.Params{
			"tenant_uuid":       tenantID,
			"feature_flag_name": ffName,
			"expectStatus":      tools.ExpectExactStatus(200),
		}, nil)

		tenantsAPI.Read(tools.Params{
			"tenant": tenantID,
			"expectPayload": func(b []byte) {
				data := tools.UnmarshalVaultResponse(b)
				ffArr := data.Get("tenant.feature_flags").Array()
				Expect(ffArr).To(HaveLen(0))
			},
		}, nil)
	})
})
