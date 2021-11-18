package project

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI   testapi.TestAPI
	ClientAPI testapi.TestAPI
)

var _ = Describe("Project", func() {
	var client model.Client

	BeforeSuite(func() {
		client = specs.CreateRandomClient(ClientAPI)
	}, 1.0)

	It("can be created", func() {
		createPayload := fixtures.RandomProjectCreatePayload()
		createPayload["tenant_uuid"] = client.UUID

		params := testapi.Params{
			"expectStatus": testapi.ExpectExactStatus(http.StatusCreated),
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
			"client": client.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("can be read", func() {
		project := specs.CreateRandomProject(TestAPI, client.UUID)
		createdData := iam_specs.ConvertToGJSON(project)

		TestAPI.Read(testapi.Params{
			"client":       project.TenantUUID,
			"project":      project.UUID,
			"expectStatus": testapi.ExpectExactStatus(http.StatusOK),
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(createdData, json.Get("project"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		project := specs.CreateRandomProject(TestAPI, client.UUID)

		TestAPI.Delete(testapi.Params{
			"expectStatus": testapi.ExpectExactStatus(http.StatusNoContent),
			"client":       project.TenantUUID,
			"project":      project.UUID,
		}, nil)

		deletedData := TestAPI.Read(testapi.Params{
			"client":       project.TenantUUID,
			"project":      project.UUID,
			"expectStatus": testapi.ExpectExactStatus(http.StatusOK),
		}, nil)
		Expect(deletedData.Get("project.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be listed", func() {
		project := specs.CreateRandomProject(TestAPI, client.UUID)

		TestAPI.List(testapi.Params{
			"client":       project.TenantUUID,
			"expectStatus": testapi.ExpectExactStatus(http.StatusOK),
			"expectPayload": func(json gjson.Result) {
				iam_specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("projects").Array(),
					iam_specs.ConvertToGJSON(project))
			},
		}, url.Values{})
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomProjectCreatePayload()
		createPayload["tenant_uuid"] = client.UUID
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := testapi.Params{
			"expectStatus": testapi.ExpectExactStatus(http.StatusCreated),
			"expectPayload": func(json gjson.Result) {
				projectData := json.Get("project")
				Expect(projectData.Map()).To(HaveKey("uuid"))
				Expect(projectData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
			"client": client.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})
})
