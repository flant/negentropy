package flant_iam_preparing

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

var serverLabels = map[string]string{"system": "ubuntu20"}

type Suite struct {
	IamVaultClient *http.Client
}

type CheckingSSHConnectionEnvironment struct {
	Tenant               model.Tenant
	User                 model.User
	Project              model.Project
	Group                model.Group
	Rolebinding          model.RoleBinding
	TestServer           specs.ServerRegistrationResult
	UserJWToken          string
	ServerLabels         map[string]string
	TestServerIdentifier string
}

func (st *Suite) BeforeSuite() {
	// try to read TEST_VAULT_SECOND_TOKEN, ROOT_VAULT_BASE_URL
	st.IamVaultClient = lib.NewConfiguredIamVaultClient()

	// try to read TEST_VAULT_SECOND_TOKEN, AUTH_VAULT_BASE_URL
	// authVaulyClient = lib.NewConfiguredIamAuthVaultClient()
}

func (st *Suite) PrepareForSSHTesting() CheckingSSHConnectionEnvironment {
	const (
		sshRole              = "ssh"
		testServerIdentifier = "test-server"
		serverRole           = "servers"
	)

	var result CheckingSSHConnectionEnvironment
	// create some tenant
	result.Tenant = specs.CreateRandomTenant(lib.NewTenantAPI(st.IamVaultClient))
	fmt.Printf("Created tenant:%#v\n", result.Tenant)

	// create some project
	result.Project = specs.CreateRandomProject(lib.NewProjectAPI(st.IamVaultClient), result.Tenant.UUID)
	fmt.Printf("Created project:%#v\n", result.Project)

	// create some user at the tenant
	result.User = specs.CreateRandomUser(lib.NewUserAPI(st.IamVaultClient), result.Tenant.UUID)
	fmt.Printf("Created user:%#v\n", result.User)

	// create a group with the user
	result.Group = specs.CreateRandomGroupWithUser(lib.NewGroupAPI(st.IamVaultClient), result.User.TenantUUID, result.User.UUID)
	fmt.Printf("Created group:%#v\n", result.Group)

	// create a role 'ssh' if not exists
	st.createRoleIfNotExist(sshRole)

	// create a role 'servers' if not exists
	st.createRoleIfNotExist(serverRole)

	// create rolebinding for a user in project with the ssh role
	result.Rolebinding = specs.CreateRoleBinding(lib.NewRoleBindingAPI(st.IamVaultClient),
		model.RoleBinding{
			TenantUUID: result.User.TenantUUID,
			Version:    "",
			Identifier: uuid.New(),
			ValidTill:  1000000,
			RequireMFA: false,
			Members:    result.Group.Members,
			AnyProject: true,
			Roles:      []model.BoundRole{{Name: sshRole, Options: map[string]interface{}{}}},
		})
	fmt.Printf("Created rolebinding:%#v\n", result.Rolebinding)

	// register as a server 'test_server'
	result.TestServer = specs.RegisterServer(lib.NewServerAPI(st.IamVaultClient),
		model2.Server{
			TenantUUID:  result.Tenant.UUID,
			ProjectUUID: result.Project.UUID,
			Identifier:  testServerIdentifier,
			Labels:      serverLabels,
		})
	fmt.Printf("Created testServer Server:%#v\n", result.TestServer)
	result.ServerLabels = serverLabels
	result.TestServerIdentifier = testServerIdentifier

	// add connection_info for a server 'test_server'
	server := specs.UpdateConnectionInfo(lib.NewConnectionInfoAPI(st.IamVaultClient),
		model2.Server{
			UUID:        result.TestServer.ServerUUID,
			TenantUUID:  result.Tenant.UUID,
			ProjectUUID: result.Project.UUID,
		},
		model2.ConnectionInfo{
			Hostname: testServerIdentifier,
		},
	)
	fmt.Printf("connection_info is updated: %#v\n", server.ConnectionInfo)

	// create and get multipass for a user
	_, result.UserJWToken = specs.CreateUserMultipass(lib.NewUserMultipassAPI(st.IamVaultClient),
		result.User, "test", 100*time.Second, 1000*time.Second, []string{"ssh"})
	fmt.Printf("user JWToken: : %#v\n", result.UserJWToken)
	return result
}

func (st Suite) createRoleIfNotExist(roleName string) {
	roleAPI := lib.NewRoleAPI(st.IamVaultClient)
	var roleNotExists bool
	rawRole := roleAPI.Read(tools.Params{
		"name":         roleName,
		"expectStatus": api.ExpectStatus("%d > 0"),
		"expectPayload": func(json gjson.Result) {
			roleNotExists = json.String() == "{\"error\":\"not found\"}"
		},
	}, nil)
	if roleNotExists {
		rawRole = roleAPI.Create(tools.Params{}, url.Values{},
			map[string]interface{}{
				"name":        roleName,
				"description": roleName,
				"scope":       model.RoleScopeProject,
			})
	}
	fmt.Printf("role: %s\n", rawRole.String())
}
