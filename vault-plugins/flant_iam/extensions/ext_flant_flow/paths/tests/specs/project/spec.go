package project

import (
	"fmt"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI   tests.TestAPI
	ClientAPI tests.TestAPI

	TenantAPI      tests.TestAPI
	RoleAPI        tests.TestAPI
	TeamAPI        tests.TestAPI
	GroupAPI       tests.TestAPI
	ConfigAPI      testapi.ConfigAPI
	RoleBindingAPI tests.TestAPI
)

var _ = Describe("Project", func() {
	var client model.Client
	var flantFlowCfg *config.FlantFlowConfig
	var devopsTeam model.Team

	BeforeSuite(func() {
		flantFlowCfg = specs.ConfigureFlantFlow(TenantAPI, RoleAPI, TeamAPI, GroupAPI, ConfigAPI)
		fmt.Printf("%#v\n", flantFlowCfg)
		client = specs.CreateRandomClient(ClientAPI)
		devopsTeam = specs.CreateDevopsTeam(TeamAPI)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomProjectAtClientWithIdentifier(client.UUID, identifier, statusCodeCondition)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := fixtures.RandomProjectCreatePayload()
		createPayload["tenant_uuid"] = client.UUID

		params := tests.Params{
			"expectStatus": tests.ExpectExactStatus(http.StatusCreated),
			"expectPayload": func(json gjson.Result) {
				projectData := json.Get("project")
				Expect(projectData.Map()).To(HaveKey("uuid"))
				Expect(projectData.Map()).To(HaveKey("tenant_uuid"))
				Expect(projectData.Map()).To(HaveKey("resource_version"))
				Expect(projectData.Map()).To(HaveKey("identifier"))
				Expect(projectData.Map()).To(HaveKey("feature_flags"))
				Expect(projectData.Map()).To(HaveKey("archiving_timestamp"))
				Expect(projectData.Map()).To(HaveKey("archiving_hash"))
				Expect(projectData.Get("uuid").String()).To(HaveLen(36))
				Expect(projectData.Get("resource_version").String()).To(HaveLen(36))
				Expect(projectData.Map()).ToNot(HaveKey("origin"))
			},
			"client": client.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	Context("client uniqueness of project identifier", func() {
		identifier := uuid.New()
		It("Can be created project with some identifier", func() {
			tryCreateRandomProjectAtClientWithIdentifier(client.UUID, identifier, "%d == 201")
		})
		It("Can not be the same identifier at the same tenant", func() {
			tryCreateRandomProjectAtClientWithIdentifier(client.UUID, identifier, "%d >= 400")
		})
		It("Can be same identifier at other tenant", func() {
			client = specs.CreateRandomClient(ClientAPI)
			tryCreateRandomProjectAtClientWithIdentifier(client.UUID, identifier, "%d == 201")
		})
	})

	var project model.Project
	It("can be created with devops service pack", func() {
		createPayload := fixtures.RandomProjectCreatePayload()
		createPayload["tenant_uuid"] = client.UUID
		createPayload["devops_team"] = devopsTeam.UUID
		createPayload["service_packs"] = []string{model.DevOps}

		params := tests.Params{
			"client":       client.UUID,
			"expectStatus": tests.ExpectExactStatus(http.StatusCreated),
			"expectPayload": func(json gjson.Result) {
				projectData := json.Get("project")
				Expect(projectData.Map()).To(HaveKey("uuid"))
				Expect(projectData.Map()).To(HaveKey("tenant_uuid"))
				Expect(projectData.Map()).To(HaveKey("resource_version"))
				Expect(projectData.Map()).To(HaveKey("identifier"))
				Expect(projectData.Map()).To(HaveKey("feature_flags"))
				Expect(projectData.Map()).To(HaveKey("archiving_timestamp"))
				Expect(projectData.Map()).To(HaveKey("archiving_hash"))
				Expect(projectData.Get("uuid").String()).To(HaveLen(36))
				Expect(projectData.Get("resource_version").String()).To(HaveLen(36))
				Expect(projectData.Map()).ToNot(HaveKey("origin"))
				project = model.Project{
					UUID:       projectData.Get("uuid").String(),
					TenantUUID: projectData.Get("tenant_uuid").String(),
					Version:    projectData.Get("resource_version").String(),
					Identifier: projectData.Get("identifier").String(),
				}
			},
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	It("after devops_service_pack rolebinding DevOps exists, and contain DirectGroup, "+
		"after changing devops_team_uuid, this rb is deleted", func() {
		rolebidingsData := RoleBindingAPI.List(tests.Params{
			"tenant": client.UUID,
		}, nil).Get("role_bindings")
		roleBindingUUID := ""
		for _, rolebindingData := range rolebidingsData.Array() {
			if rolebindingData.Get("description").String() == "DevOps" && rolebindingData.Get("archiving_timestamp").String() == "0" {
				for _, grData := range rolebindingData.Get("groups").Array() {
					if grData.String() == devopsTeam.Groups[0].GroupUUID {
						roleBindingUUID = rolebindingData.Get("uuid").String()
						break
					}
				}
			}
		}
		Expect(roleBindingUUID).ToNot(BeEmpty())
		devopsTeam2 := specs.CreateDevopsTeam(TeamAPI)
		updatePayload := map[string]interface{}{
			"devops_team":      devopsTeam2.UUID,
			"resource_version": project.Version,
			"service_packs":    []string{model.DevOps},
			"identifier":       project.Identifier,
		}
		TestAPI.Update(tests.Params{
			"client":       project.TenantUUID,
			"project":      project.UUID,
			"expectStatus": tests.ExpectExactStatus(200),
		}, nil, updatePayload)
		rolebidingData := RoleBindingAPI.Read(tests.Params{
			"tenant":       client.UUID,
			"role_binding": roleBindingUUID,
		}, nil).Get("role_binding")
		Expect(rolebidingData.Get("archiving_timestamp").String()).ToNot(Equal("0"))
	})

	It("can be read", func() {
		project := specs.CreateRandomProject(TestAPI, client.UUID)
		createdData := iam_specs.ConvertToGJSON(project)
		project.Origin = consts.OriginFlantFlow
		TestAPI.Read(tests.Params{
			"client":       project.TenantUUID,
			"project":      project.UUID,
			"expectStatus": tests.ExpectExactStatus(http.StatusOK),
			"expectPayload": func(json gjson.Result) {
				iam_specs.IsSubsetExceptKeys(createdData, json.Get("project"), "extensions")
				Expect(json.Get("project").Map()).ToNot(HaveKey("origin"))
			},
		}, nil)
	})

	It("can be listed", func() {
		project := specs.CreateRandomProject(TestAPI, client.UUID)

		TestAPI.List(tests.Params{
			"client":       project.TenantUUID,
			"expectStatus": tests.ExpectExactStatus(http.StatusOK),
			"expectPayload": func(json gjson.Result) {
				projectsArray := json.Get("projects").Array()
				iam_specs.CheckArrayContainsElementByUUIDExceptKeys(projectsArray,
					iam_specs.ConvertToGJSON(project), "extensions", "service_packs")
				Expect(len(projectsArray)).To(BeNumerically(">", 0))
				Expect(projectsArray[0].Map()).ToNot(HaveKey("origin"))
			},
		}, url.Values{})
	})

	It("can be updated", func() {
		project := specs.CreateRandomProject(TestAPI, client.UUID)

		updatePayload := fixtures.RandomProjectCreatePayload()
		updatePayload["tenant_uuid"] = project.TenantUUID
		updatePayload["resource_version"] = project.Version
		TestAPI.Update(tests.Params{
			"client":       project.TenantUUID,
			"project":      project.UUID,
			"expectStatus": tests.ExpectExactStatus(200),
		}, nil, updatePayload)

		TestAPI.Read(tests.Params{
			"client":  project.TenantUUID,
			"project": project.UUID,
			"expectPayload": func(json gjson.Result) {
				projectData := json.Get("project")
				iam_specs.IsMapSubsetOfSetExceptKeys(updatePayload, projectData, "archiving_timestamp",
					"archiving_hash", "uuid", "resource_version", "origin", "tenant_uuid", "service_packs")
				Expect(projectData.Map()).ToNot(HaveKey("origin"))
			},
		}, nil)
	})

	It("can be deleted", func() {
		project := specs.CreateRandomProject(TestAPI, client.UUID)

		deleteProject(project)

		deletedData := TestAPI.Read(tests.Params{
			"client":       project.TenantUUID,
			"project":      project.UUID,
			"expectStatus": tests.ExpectExactStatus(http.StatusOK),
		}, nil)
		Expect(deletedData.Get("project.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	It("can be created with privileged", func() {
		createPayload := fixtures.RandomProjectCreatePayload()
		createPayload["tenant_uuid"] = client.UUID
		originalUUID := uuid.New()
		createPayload["uuid"] = originalUUID

		params := tests.Params{
			"expectStatus": tests.ExpectExactStatus(http.StatusCreated),
			"expectPayload": func(json gjson.Result) {
				projectData := json.Get("project")
				Expect(projectData.Map()).To(HaveKey("uuid"))
				Expect(projectData.Map()["uuid"].String()).To(Equal(originalUUID))
				Expect(projectData.Map()).ToNot(HaveKey("origin"))
			},
			"client": client.UUID,
		}
		TestAPI.CreatePrivileged(params, url.Values{}, createPayload)
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			project := specs.CreateRandomProject(TestAPI, client.UUID)
			deleteProject(project)

			TestAPI.Delete(tests.Params{
				"client":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil)
		})

		It("can't be updated", func() {
			project := specs.CreateRandomProject(TestAPI, client.UUID)
			deleteProject(project)

			updatePayload := fixtures.RandomProjectCreatePayload()
			updatePayload["tenant_uuid"] = project.TenantUUID
			updatePayload["resource_version"] = project.Version
			TestAPI.Update(tests.Params{
				"client":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil, updatePayload)
		})
	})

	Context("restoring deleted project", func() {
		It("can be restored after deleting", func() {
			project := specs.CreateRandomProject(TestAPI, client.UUID)
			deleteProject(project)

			TestAPI.Restore(tests.Params{
				"client":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": tests.ExpectExactStatus(200),
				"expectPayload": func(json gjson.Result) {
					projectData := json.Get("project")
					Expect(projectData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically("==", int64(0))))
				},
			}, nil)
		})

		It("cant be restored after deleting client", func() {
			otherClient := specs.CreateRandomClient(ClientAPI)
			project := specs.CreateRandomProject(TestAPI, otherClient.UUID)
			deleteProject(project)
			ClientAPI.Delete(tests.Params{
				"expectStatus": tests.ExpectExactStatus(http.StatusNoContent),
				"client":       otherClient.UUID,
			}, nil)

			TestAPI.Restore(tests.Params{
				"client":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": tests.ExpectExactStatus(400),
			}, nil)

			TestAPI.Read(tests.Params{
				"client":       project.TenantUUID,
				"project":      project.UUID,
				"expectStatus": tests.ExpectExactStatus(200),
				"expectPayload": func(json gjson.Result) {
					projectData := json.Get("project")
					Expect(projectData.Get("archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
				},
			}, nil)
		})
	})
})

func tryCreateRandomProjectAtClientWithIdentifier(clientUUID string,
	projectIdentifier interface{}, statusCodeCondition string) {
	payload := fixtures.RandomProjectCreatePayload()
	payload["identifier"] = projectIdentifier

	params := tests.Params{
		"client":       clientUUID,
		"expectStatus": tests.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}

func deleteProject(project model.Project) {
	TestAPI.Delete(tests.Params{
		"expectStatus": tests.ExpectExactStatus(http.StatusNoContent),
		"client":       project.TenantUUID,
		"project":      project.UUID,
	}, nil)
}
