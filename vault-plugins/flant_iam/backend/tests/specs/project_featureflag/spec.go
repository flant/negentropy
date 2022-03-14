package project_featureflag

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
)

var (
	TenantAPI      api.TestAPI
	ProjectAPI     api.TestAPI
	FeatureFlagAPI api.TestAPI
	TestAPI        api.TestAPI
)

var _ = Describe("Project feature flags", func() {
	var tenantID, ffName, projectID string
	BeforeSuite(func() {
		res := TenantAPI.Create(nil, url.Values{}, fixtures.RandomTenantCreatePayload())
		tenantID = res.Get("tenant.uuid").String()
		res = ProjectAPI.Create(api.Params{"tenant": tenantID}, url.Values{}, fixtures.RandomProjectCreatePayload())
		projectID = res.Get("project.uuid").String()
		res = FeatureFlagAPI.Create(api.Params{"tenant": tenantID}, url.Values{}, fixtures.RandomFeatureFlagCreatePayload())
		ffName = res.Get("feature_flag.name").String()
	})

	It("can be bound", func() {
		params := api.Params{
			"expectStatus":      api.ExpectExactStatus(200),
			"tenant":            tenantID,
			"project":           projectID,
			"feature_flag_name": ffName,
		}

		data := map[string]interface{}{}

		_ = TestAPI.Create(params, url.Values{}, data)
	})

	It("can be read from project", func() {
		ProjectAPI.Read(api.Params{
			"tenant":  tenantID,
			"project": projectID,
			"expectPayload": func(json gjson.Result) {
				ffArr := json.Get("project.feature_flags").Array()
				Expect(ffArr).To(HaveLen(1))
				Expect(ffArr[0].String()).To(Equal(ffName))
			},
		}, nil)
	})

	It("can be unbound", func() {
		TestAPI.Delete(api.Params{
			"tenant":            tenantID,
			"project":           projectID,
			"feature_flag_name": ffName,
			"expectStatus":      api.ExpectExactStatus(200),
		}, nil)

		ProjectAPI.Read(api.Params{
			"tenant":  tenantID,
			"project": projectID,
			"expectPayload": func(json gjson.Result) {
				ffArr := json.Get("tenant.feature_flags").Array()
				Expect(ffArr).To(HaveLen(0))
			},
		}, nil)
	})

	Context("after deletion project", func() {
		It("can't be deleted", func() {
			ProjectAPI.Delete(api.Params{
				"expectStatus": api.ExpectExactStatus(http.StatusNoContent),
				"tenant":       tenantID,
				"project":      projectID,
			}, nil)

			TestAPI.Delete(api.Params{
				"tenant":            tenantID,
				"project":           projectID,
				"feature_flag_name": ffName,
				"expectStatus":      api.ExpectExactStatus(400),
			}, nil)
		})
	})
})
