package project

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

var (
	TestAPI   api.TestAPI
	TenantAPI api.TestAPI
)

var _ = Describe("Group", func() {
	var tenant model.Tenant

	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
	}, 1.0)

	It("can be created", func() {
		createPayload := fixtures.RandomGroupCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID

		params := api.Params{
			"expectStatus": api.ExpectExactStatus(http.StatusCreated),
			"expectPayload": func(json gjson.Result) {
				projectData := json.Get("project")
				Expect(projectData.Map()).To(HaveKey("uuid"))
				Expect(projectData.Map()).To(HaveKey("tenant_uuid"))
				Expect(projectData.Map()).To(HaveKey("resource_version"))
				Expect(projectData.Map()).To(HaveKey("identifier"))
				Expect(projectData.Map()).To(HaveKey("feature_flags"))
				Expect(projectData.Map()).To(HaveKey("archiving_timestamp"))
				Expect(projectData.Map()).To(HaveKey("archiving_hash"))
				Expect(projectData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(projectData.Get("resource_version").String()).ToNot(HaveLen(10))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		project := specs.CreateRandomProject(TestAPI, tenant.UUID)
		createdData := specs.ConvertToGJSON(project)

		TestAPI.Read(api.Params{
			"tenant":       project.TenantUUID,
			"project":      project.UUID,
			"expectStatus": api.ExpectExactStatus(http.StatusOK),
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(createdData, json.Get("project"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		project := specs.CreateRandomProject(TestAPI, tenant.UUID)

		TestAPI.Delete(api.Params{
			"expectStatus": api.ExpectExactStatus(http.StatusNoContent),
			"tenant":       project.TenantUUID,
			"project":      project.UUID,
		}, nil)

		deletedData := TestAPI.Read(api.Params{
			"tenant":       project.TenantUUID,
			"project":      project.UUID,
			"expectStatus": api.ExpectExactStatus(http.StatusOK),
		}, nil)
		Expect(deletedData.Get("project.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		project := specs.CreateRandomProject(TestAPI, tenant.UUID)

		TestAPI.List(api.Params{
			"tenant":       project.TenantUUID,
			"expectStatus": api.ExpectExactStatus(http.StatusOK),
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("projects").Array(),
					specs.ConvertToGJSON(project))
			},
		}, url.Values{})
	})

	It("can be created with priveleged", func() {
		createPayload := fixtures.RandomGroupCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := api.Params{
			"expectStatus": api.ExpectExactStatus(http.StatusCreated),
			"expectPayload": func(json gjson.Result) {
				groupData := json.Get("project")
				Expect(groupData.Map()).To(HaveKey("uuid"))
				Expect(groupData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})
})
