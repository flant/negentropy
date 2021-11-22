package server

import (
	"fmt"
	"net/url"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

var (
	TestAPI    api.TestAPI
	TenantAPI  api.TestAPI
	ProjectAPI api.TestAPI
	RoleAPI    api.TestAPI
	ConfigAPI  api.ConfigAPI
)

var _ = Describe("Server", func() {
	var (
		tenant  model.Tenant
		project model.Project
	)

	BeforeSuite(func() {
		ConfigAPI.GenerateCSR()
		ConfigAPI.ConfigureKafka("cert", []string{"192.168.1.1:9093"})
		ConfigAPI.EnableJWT()

		tenant = specs.CreateRandomTenant(TenantAPI)
		fmt.Printf("%#v\n", tenant)

		project = specs.CreateRandomProject(ProjectAPI, tenant.UUID)
		fmt.Printf("%#v\n", project)

		sshRoleName := "ssh"
		createRoleForExtServAccess(sshRoleName)
		serversRoleName := "servers"
		createRoleForExtServAccess(serversRoleName)

		cfg := map[string]interface{}{
			"roles_for_servers":                    serversRoleName,
			"role_for_ssh_access":                  sshRoleName,
			"name":                                 sshRoleName,
			"delete_expired_password_seeds_after":  "1000000",
			"expire_password_seed_after_reveal_in": "1000000",
			"last_allocated_uid":                   "10000",
		}
		ConfigAPI.ConfigureExtensionServerAccess(cfg)
	}, 1.0)

	It("can be created", func() {
		createPayload := api.Params{
			"identifier": "testServerIdentifier",
			"labels":     map[string]string{"system": "ubuntu20"},
		}
		params := api.Params{
			"expectStatus": api.ExpectExactStatus(200),
			"expectPayload": func(json gjson.Result) {
				fmt.Printf("%#v", json)
				// userData := json.Get("group")
				Expect(json.Map()).To(HaveKey("multipassJWT"))
				Expect(json.Map()).To(HaveKey("uuid"))
			},
			"tenant":  tenant.UUID,
			"project": project.UUID,
		}
		TestAPI.Create(params, url.Values{}, createPayload)
	})
})

func createRoleForExtServAccess(sshRoleName string) {
	var roleNotExists bool
	RoleAPI.Read(api.Params{
		"name":         sshRoleName,
		"expectStatus": api.ExpectStatus("%d > 0"),
		"expectPayload": func(json gjson.Result) {
			roleNotExists = json.String() == "{\"error\":\"not found\"}"
		},
	}, nil)
	if roleNotExists {
		RoleAPI.Create(api.Params{}, url.Values{},
			map[string]interface{}{
				"name":        sshRoleName,
				"description": sshRoleName,
				"scope":       model.RoleScopeProject,
			})
	}
}
