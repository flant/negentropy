package project

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI   api.TestAPI
	TenantAPI api.TestAPI
)

var _ = Describe("Project", func() {
	var tenant model.Tenant

	BeforeSuite(func() {
		tenant = specs.CreateRandomTenant(TenantAPI)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomGroupAtTenantWithIdentifier(tenant.UUID, identifier, statusCodeCondition)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomProjectCreatePayload()
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
				Expect(projectData.Map()).To(HaveKey("origin"))
				Expect(projectData.Get("origin").String()).To(Equal("iam"))
				Expect(projectData.Get("uuid").String()).ToNot(HaveLen(10))
				Expect(projectData.Get("resource_version").String()).ToNot(HaveLen(10))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	Context("tenant uniqueness of project identifier", func() {
		identifier := uuid.New()
		It("Can be created project with some identifier", func() {
			tryCreateRandomGroupAtTenantWithIdentifier(tenant.UUID, identifier, "%d == 201")
		})
		It("Can not be the same identifier at the same tenant", func() {
			tryCreateRandomGroupAtTenantWithIdentifier(tenant.UUID, identifier, "%d >= 400")
		})
		It("Can be same identifier at other tenant", func() {
			tenant = specs.CreateRandomTenant(TenantAPI)
			tryCreateRandomGroupAtTenantWithIdentifier(tenant.UUID, identifier, "%d == 201")
		})
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

	It("can be updated", func() {
		project := specs.CreateRandomProject(TestAPI, tenant.UUID)
		updatePayload := fixtures.RandomProjectCreatePayload()
		updatePayload["tenant_uuid"] = project.TenantUUID
		updatePayload["resource_version"] = project.Version

		var updateData gjson.Result
		TestAPI.Update(api.Params{
			"tenant":  project.TenantUUID,
			"project": project.UUID,
			"expectPayload": func(json gjson.Result) {
				updateData = json
				projectData := updateData.Get("project")
				Expect(projectData.Map()).To(HaveKey("origin"))
				Expect(projectData.Get("origin").String()).To(Equal("iam"))
			},
		}, nil, updatePayload)

		TestAPI.Read(api.Params{
			"tenant":  project.TenantUUID,
			"project": project.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.IsSubsetExceptKeys(updateData, json, "full_restore")
			},
		}, nil)
	})

	It("can be deleted", func() {
		project := specs.CreateRandomProject(TestAPI, tenant.UUID)

		deleteProject(project)

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

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomProjectCreatePayload()
		createPayload["tenant_uuid"] = tenant.UUID
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := api.Params{
			"expectStatus": api.ExpectExactStatus(http.StatusCreated),
			"expectPayload": func(json gjson.Result) {
				projectData := json.Get("project")
				Expect(projectData.Map()).To(HaveKey("uuid"))
				Expect(projectData.Map()["uuid"].String()).To(Equal(originalUUID))
			},
			"tenant": tenant.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("restoring deleted project", func() {
		It("can be restored after deleting", func() {
			project := specs.CreateRandomProject(TestAPI, tenant.UUID)
			deleteProject(project)
			deletedData := TestAPI.Read(api.Params{
				"tenant":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": api.ExpectExactStatus(http.StatusOK),
			}, nil)
			Expect(deletedData.Get("project.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))

			restoreData := TestAPI.Restore(api.Params{
				"tenant":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": api.ExpectExactStatus(http.StatusOK),
			}, nil)
			Expect(restoreData.Get("project.archiving_timestamp").Int()).To(Equal(int64(0)))
		})

		It("cant be restored after deleting client", func() {
			otherClient := specs.CreateRandomTenant(TenantAPI)
			project := specs.CreateRandomProject(TestAPI, otherClient.UUID)
			deleteProject(project)
			TenantAPI.Delete(api.Params{
				"expectStatus": api.ExpectExactStatus(http.StatusNoContent),
				"tenant":       otherClient.UUID,
			}, nil)

			TestAPI.Restore(api.Params{
				"tenant":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)

			TestAPI.Read(api.Params{
				"tenant":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": api.ExpectExactStatus(200),
				"expectPayload": func(json gjson.Result) {
					projectData := json.Get("project")
					Expect(projectData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
				},
			}, nil)
		})
	})
	Context("after deletion", func() {
		It("can't be deleted", func() {
			project := specs.CreateRandomProject(TestAPI, tenant.UUID)
			deleteProject(project)

			TestAPI.Delete(api.Params{
				"tenant":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			project := specs.CreateRandomProject(TestAPI, tenant.UUID)
			deleteProject(project)
			updatePayload := fixtures.RandomProjectCreatePayload()
			updatePayload["tenant_uuid"] = project.TenantUUID
			updatePayload["resource_version"] = project.Version

			TestAPI.Update(api.Params{
				"tenant":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": api.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})
})

func tryCreateRandomGroupAtTenantWithIdentifier(tenantUUID,
	projectIdentifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomProjectCreatePayload()
	payload["identifier"] = projectIdentifier

	params := api.Params{
		"tenant":       tenantUUID,
		"expectStatus": api.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}

func deleteProject(project model.Project) {
	TestAPI.Delete(api.Params{
		"expectStatus": api.ExpectExactStatus(http.StatusNoContent),
		"tenant":       project.TenantUUID,
		"project":      project.UUID,
	}, nil)
}
