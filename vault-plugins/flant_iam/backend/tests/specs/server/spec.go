package server

import (
	"fmt"
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	cfg_api "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var (
	TestAPI    api.TestAPI
	TenantAPI  api.TestAPI
	ProjectAPI api.TestAPI
	RoleAPI    api.TestAPI
	ConfigAPI  cfg_api.ConfigAPI
)

var _ = Describe("Server", func() {
	var (
		tenant           model.Tenant
		project          model.Project
		serverUUID       ext_model.ServerUUID
		serverCreateData gjson.Result
	)

	BeforeSuite(func() {
		ConfigAPI.ConfigureKafka("cert", []string{"192.168.1.1:9093"})
		ConfigAPI.EnableJWT()

		tenant = specs.CreateRandomTenant(TenantAPI)
		fmt.Printf("%#v\n", tenant)

		project = specs.CreateRandomProject(ProjectAPI, tenant.UUID)
		fmt.Printf("%#v\n", project)

		sshRoleName := "ssh.open"
		createRoleForExtServAccess(sshRoleName)
		serverRoleName := "server"
		createRoleForExtServAccess(serverRoleName)

		cfg := map[string]interface{}{
			"roles_for_servers":                    serverRoleName,
			"role_for_ssh_access":                  sshRoleName,
			"name":                                 sshRoleName,
			"delete_expired_password_seeds_after":  "1000000",
			"expire_password_seed_after_reveal_in": "1000000",
			"last_allocated_uid":                   "10000",
		}
		ConfigAPI.ConfigureExtensionServerAccess(cfg)
	}, 1.0)

	Describe("payload", func() {
		DescribeTable("identifier",
			func(identifier interface{}, statusCodeCondition string) {
				tryCreateRandomServerAtTenantAndProjectWithIdentifier(tenant.UUID, project.UUID, identifier, statusCodeCondition)
			},
			Entry("hyphen, symbols and numbers are allowed", uuid.New(), "%d == 201"),
			Entry("under_score allowed", "under_score"+uuid.New(), "%d == 201"),
			Entry("russian symbols forbidden", "РусскийТекст", "%d >= 400"),
			Entry("space forbidden", "invalid space", "%d >= 400"),
		)
	})

	It("can be created", func() {
		createPayload := api.Params{
			"identifier": "testServerIdentifier",
			"labels":     map[string]string{"system": "ubuntu20"},
		}
		params := api.Params{
			"expectStatus": api.ExpectExactStatus(201),
			"expectPayload": func(json gjson.Result) {
				Expect(json.Map()).To(HaveKey("multipassJWT"))
				Expect(json.Map()).To(HaveKey("uuid"))
				serverUUID = json.Get("uuid").String()
			},
			"tenant":  tenant.UUID,
			"project": project.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})

	Context("project uniqueness of server identifier", func() {
		identifier := uuid.New()
		It("Can be created server with some identifier", func() {
			tryCreateRandomServerAtTenantAndProjectWithIdentifier(tenant.UUID, project.UUID, identifier, "%d == 201")
		})
		It("Can not be the same identifier at the same tenant & project", func() {
			tryCreateRandomServerAtTenantAndProjectWithIdentifier(tenant.UUID, project.UUID, identifier, "%d >= 400")
		})
		It("Can be same identifier at other project", func() {
			otherProject := specs.CreateRandomProject(ProjectAPI, tenant.UUID)
			tryCreateRandomServerAtTenantAndProjectWithIdentifier(tenant.UUID, otherProject.UUID, identifier, "%d == 201")
		})
	})

	It("can be read", func() {
		TestAPI.Read(api.Params{
			"tenant":       project.TenantUUID,
			"project":      project.UUID,
			"server":       serverUUID,
			"expectStatus": api.ExpectExactStatus(http.StatusOK),
			"expectPayload": func(json gjson.Result) {
				serverData := json.Get("server")
				for _, k := range []string{
					"archiving_timestamp", "archiving_hash", "uuid", "tenant_uuid",
					"project_uuid", "resource_version", "identifier", "multipass_uuid", "fingerprint", "labels",
					"annotations", "connection_info",
				} {
					Expect(serverData.Map()).To(HaveKey(k))
				}
				Expect(serverData.Get("archiving_timestamp").Int()).To(Equal(int64(0)))
				Expect(serverData.Get("archiving_hash").Int()).To(Equal(int64(0)))
				Expect(serverData.Get("uuid").String()).To(HaveLen(36))
				Expect(serverData.Get("tenant_uuid").String()).To(Equal(tenant.UUID))
				Expect(serverData.Get("project_uuid").String()).To(Equal(project.UUID))
				Expect(serverData.Get("resource_version").String()).To(HaveLen(36))
				Expect(serverData.Get("identifier").String()).To(Equal("testServerIdentifier"))
				Expect(serverData.Get("multipass_uuid").String()).To(HaveLen(36))
				Expect(serverData.Get("labels").Map()).To(HaveLen(1))
				Expect(serverData.Get("annotations").String()).To(Equal("{}"))
				Expect(serverData.Get("connection_info").Map()).To(HaveLen(4))
				serverCreateData = serverData
			},
		}, nil)
	})

	It("can be listed", func() {
		TestAPI.List(api.Params{
			"tenant":  tenant.UUID,
			"project": project.UUID,
			"expectPayload": func(json gjson.Result) {
				specs.CheckArrayContainsElementByUUIDExceptKeys(json.Get("servers").Array(),
					serverCreateData)
			},
		}, url.Values{})
	})

	It("can be updated", func() {
		updatePayload := api.Params{
			"identifier":       "testServerIdentifierUpdated",
			"labels":           map[string]string{"system": "ubuntu20", "type": "metal"},
			"resource_version": serverCreateData.Get("resource_version").String(),
		}

		TestAPI.Update(api.Params{
			"tenant":  tenant.UUID,
			"project": project.UUID,
			"server":  serverUUID,
			"expectPayload": func(json gjson.Result) {
				serverData := json.Get("server")
				for _, k := range []string{
					"archiving_timestamp", "archiving_hash", "uuid", "tenant_uuid",
					"project_uuid", "resource_version", "identifier", "multipass_uuid", "fingerprint", "labels",
					"annotations", "connection_info",
				} {
					Expect(serverData.Map()).To(HaveKey(k))
				}
				Expect(serverData.Get("identifier").String()).To(Equal("testServerIdentifierUpdated"))
				Expect(serverData.Get("labels").Map()).To(HaveLen(2))
			},
		}, nil, updatePayload)
	})

	It("can be deleted", func() {
		// TODO fix bug
		// TestAPI.Delete(api.Params{
		//	"tenant":  tenant.UUID,
		//	"project": project.UUID,
		//	"server":  serverUUID,
		// }, nil)
		//
		// deletedData := TestAPI.Read(api.Params{
		//	"tenant":       tenant.UUID,
		//	"project":      project.UUID,
		//	"server":       serverUUID,
		//	"expectStatus": api.ExpectExactStatus(200),
		// }, nil)
		// Expect(deletedData.Get("group.archiving_timestamp").Int()).To(SatisfyAll(BeNumerically(">", 0)))
	})

	Context("after deletion", func() {
		It("can't be deleted", func() {
			// TODO fix delete first
		})

		It("can't be updated", func() {
			// TODO fix delete first
		})
	})
})

func createRoleForExtServAccess(roleName string) {
	var roleNotExists bool
	RoleAPI.Read(api.Params{
		"name": roleName,
		"expectStatus": func(status int) {
			if status != 200 {
				roleNotExists = true
			}
		},
	}, nil)

	if roleNotExists {
		RoleAPI.Create(api.Params{}, url.Values{},
			map[string]interface{}{
				"name":        roleName,
				"description": roleName,
				"scope":       model.RoleScopeProject,
			})
	}
}

func tryCreateRandomServerAtTenantAndProjectWithIdentifier(tenantUUID string, projectUUID string,
	serverIdentifier interface{}, statusCodeCondition string) {
	payload := api.Params{
		"identifier": serverIdentifier,
		"labels":     map[string]string{"system": "ubuntu20"},
	}

	params := api.Params{
		"tenant":       tenantUUID,
		"project":      projectUUID,
		"expectStatus": api.ExpectStatus(statusCodeCondition),
	}

	TestAPI.Create(params, nil, payload)
}
