package tenant_featureflag

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TenantAPI      api.TestAPI
	FeatureFlagAPI api.TestAPI
	TestAPI        api.TestAPI
)

var _ = Describe("Tenant feature flags", func() {
	var tenantID, ffName string
	BeforeSuite(func() {
		res := TenantAPI.Create(nil, url.Values{}, fixtures.RandomTenantCreatePayload())
		tenantID = res.Get("tenant.uuid").String()
		res = FeatureFlagAPI.Create(api.Params{"tenant_uuid": tenantID}, url.Values{}, fixtures.RandomFeatureFlagCreatePayload())
		ffName = res.Get("feature_flag.name").String()
	})

	It("can be bound", func() {
		params := api.Params{
			"expectStatus":      api.ExpectExactStatus(200),
			"tenant":            tenantID,
			"feature_flag_name": ffName,
		}

		data := map[string]interface{}{
			"required_votes":   3,
			"users":            []string{uuid.New()},
			"groups":           []string{uuid.New()},
			"service_accounts": []string{uuid.New()},
		}

		_ = TestAPI.Create(params, url.Values{}, data)
	})

	It("can be read from tenant", func() {
		TenantAPI.Read(api.Params{
			"tenant": tenantID,
			"expectPayload": func(json gjson.Result) {
				ffArr := json.Get("tenant.feature_flags").Array()
				Expect(ffArr).To(HaveLen(1))
				Expect(ffArr[0].Get("name").String()).To(Equal(ffName))
				Expect(ffArr[0].Get("enabled_for_new").Bool()).To(BeTrue())
			},
		}, nil)
	})

	It("can be unbound", func() {
		TestAPI.Delete(api.Params{
			"tenant":            tenantID,
			"feature_flag_name": ffName,
			"expectStatus":      api.ExpectExactStatus(200),
		}, nil)

		TenantAPI.Read(api.Params{
			"tenant": tenantID,
			"expectPayload": func(json gjson.Result) {
				ffArr := json.Get("tenant.feature_flags").Array()
				Expect(ffArr).To(HaveLen(0))
			},
		}, nil)
	})

	Context("after deletion tenant", func() {
		It("can't be deleted", func() {
			TenantAPI.Delete(api.Params{
				"expectStatus": api.ExpectExactStatus(http.StatusNoContent),
				"tenant":       tenantID,
			}, nil)

			TestAPI.Delete(api.Params{
				"tenant":            tenantID,
				"feature_flag_name": ffName,
				"expectStatus":      api.ExpectExactStatus(400),
			}, nil)
		})
	})
})
