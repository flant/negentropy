package flant_iam_preparing

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/cli/pkg"
	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

var serverLabels = map[string]string{"system": "ubuntu20"}

type Suite struct {
	IamVaultClient *http.Client
	// IamAuthVaultClient *http.Client
}

const FlantTenantUUID = "b2c3d385-6bc7-43ff-9e75-441330442b1e" // created by start.sh
const FlantTenantID = "flant"

type CheckingEnvironment struct {
	Tenant                               model.Tenant
	ServiceAccount                       model.ServiceAccount // for testing sapassword & role register_server
	ServiceAccountPassword               model.ServiceAccountPassword
	ServiceAccountRoleBinding            model.RoleBinding
	Project                              model.Project
	User                                 model.User
	Group                                model.Group
	UserRolebinding                      model.RoleBinding
	TestServerServiceAccountMultipassJWT model.MultipassJWT
	TestServer                           model2.Server
	UserMultipassJWT                     model.MultipassJWT
}

func (st *Suite) BeforeSuite() {
	// try to read ROOT_VAULT_TOKEN, ROOT_VAULT_URL
	st.IamVaultClient = lib.NewConfiguredIamVaultClient()

	// try to read AUTH_VAULT_TOKEN, AUTH_VAULT_URL
	// st.IamAuthVaultClient = lib.NewConfiguredIamAuthVaultClient()
}

const (
	RegisterServerRole = "register_server"
	IamAuthRead        = "iam_auth_read"
	IamReadRole        = "iam_read"
)

func (st *Suite) PrepareForLoginTesting() CheckingEnvironment {
	var result CheckingEnvironment
	// create some tenant
	result.Tenant = specs.CreateRandomTenant(lib.NewTenantAPI(st.IamVaultClient))
	fmt.Printf("Created tenant:%#v\n", result.Tenant)
	// create some SA at the tenant for server registration
	result.ServiceAccount = specs.CreateRandomServiceAccount(lib.NewServiceAccountAPI(st.IamVaultClient), result.Tenant.UUID)
	fmt.Printf("Created serviceAccount:%#v\n", result.ServiceAccount)
	// create  SA password for SA login
	result.ServiceAccountPassword = specs.CreateServiceAccountPassword(lib.NewServiceAccountPasswordAPI(st.IamVaultClient),
		result.ServiceAccount, "test", 100*time.Second, []string{RegisterServerRole, IamReadRole})
	fmt.Printf("Created serviceAccountPassword:%#v\n", result.ServiceAccountPassword)

	// create a role 'IamAuthRead' if not exists
	st.createRoleIfNotExist(IamAuthRead, model.RoleScopeTenant)

	// create some project
	result.Project = specs.CreateRandomProject(lib.NewProjectAPI(st.IamVaultClient), result.Tenant.UUID)
	fmt.Printf("Created project:%#v\n", result.Project)

	// create rolebinding for a sa in project with the RegisterServerRole
	result.ServiceAccountRoleBinding = specs.CreateRoleBinding(lib.NewRoleBindingAPI(st.IamVaultClient),
		model.RoleBinding{
			TenantUUID:  result.Tenant.UUID,
			Version:     "",
			Description: "flant_iam_preparing for e2e login testing",
			ValidTill:   10_000_000_000,
			RequireMFA:  false,
			Members: []model.MemberNotation{{
				Type: model.ServiceAccountType,
				UUID: result.ServiceAccount.UUID,
			}},
			Projects:   []string{result.Project.UUID},
			AnyProject: false,
			Roles: []model.BoundRole{{Name: RegisterServerRole, Options: map[string]interface{}{}},
				{Name: IamAuthRead, Options: map[string]interface{}{}}},
		})
	fmt.Printf("Created rolebinding:%#v\n", result.ServiceAccountRoleBinding)

	// create some user at the tenant
	result.User = specs.CreateRandomUser(lib.NewUserAPI(st.IamVaultClient), result.Tenant.UUID)
	fmt.Printf("Created user:%#v\n", result.User)
	return result
}

const TestServerIdentifier = "test-server"

func (st *Suite) PrepareForSSHTesting() CheckingEnvironment {
	const (
		sshRole    = "ssh"
		serverRole = "servers"
	)
	result := st.PrepareForLoginTesting()

	err := st.WaitPrepareForLoginTesting(result, 40)
	Expect(err).ToNot(HaveOccurred())

	// create a group with the user
	result.Group = specs.CreateRandomGroupWithUser(lib.NewGroupAPI(st.IamVaultClient), result.User.TenantUUID, result.User.UUID)
	fmt.Printf("Created group:%#v\n", result.Group)

	// create rolebinding for a user in project with the ssh role
	result.UserRolebinding = specs.CreateRoleBinding(lib.NewRoleBindingAPI(st.IamVaultClient),
		model.RoleBinding{
			TenantUUID:  result.User.TenantUUID,
			Version:     "",
			Description: "flant_iam_preparing for e2e ssh testing",
			ValidTill:   10_000_000_000,
			RequireMFA:  false,
			Members:     result.Group.Members,
			Projects:    []string{result.Project.UUID},
			AnyProject:  false,
			Roles:       []model.BoundRole{{Name: sshRole, Options: map[string]interface{}{"max_ttl": "1600m", "ttl": "800m"}}},
		})
	fmt.Printf("Created rolebinding:%#v\n", result.UserRolebinding)

	// register as a server 'test_server' using serviceAccount & add connection info
	result.TestServer, result.TestServerServiceAccountMultipassJWT = registerServer(result.ServiceAccountPassword, result.Tenant.Identifier, result.Project.Identifier, TestServerIdentifier, serverLabels)

	fmt.Printf("Created testServer Server:%#v\n", result.TestServer)

	// create and get multipass for a user
	_, result.UserMultipassJWT = specs.CreateUserMultipass(lib.NewUserMultipassAPI(st.IamVaultClient),
		result.User, "test", 100*time.Second, 1000*time.Second, []string{"ssh"})
	fmt.Printf("user JWToken: : %#v\n", result.UserMultipassJWT)
	return result
}

func registerServer(saPassword model.ServiceAccountPassword, tIdentifier string, pIdentifier string,
	sIdentifier string, labels map[string]string) (model2.Server, model.MultipassJWT) {
	// get client just for tenants list
	cl := getClientAuthorizedWithSAPass(saPassword, nil)
	tenant, err := cl.GetTenantByIdentifier(tIdentifier)
	Expect(err).ToNot(HaveOccurred())
	// get client just for project list
	cl = getClientAuthorizedWithSAPass(saPassword, []map[string]interface{}{
		{"role": "iam_auth_read", "tenant_uuid": tenant.UUID}})
	project, err := cl.GetProjectByIdentifier(tenant.UUID, pIdentifier)
	Expect(err).ToNot(HaveOccurred())
	// get client for registering server
	cl = getClientAuthorizedWithSAPass(saPassword, []map[string]interface{}{
		{"role": "register_server", "tenant_uuid": tenant.UUID, "project_uuid": project.UUID}})
	serverUUID, multipassJWT, err := cl.RegisterServer(
		model2.Server{
			TenantUUID:  tenant.UUID,
			ProjectUUID: project.UUID,
			Identifier:  sIdentifier,
			Labels:      labels,
		})
	Expect(err).ToNot(HaveOccurred())
	server, err := cl.UpdateServerConnectionInfo(tenant.UUID, project.UUID, serverUUID, model2.ConnectionInfo{
		Hostname: sIdentifier,
	})
	Expect(err).ToNot(HaveOccurred())
	return *server, multipassJWT
}

func getClientAuthorizedWithSAPass(password model.ServiceAccountPassword, roles []map[string]interface{}) pkg.VaultClient {
	cl := configure.GetClientWithToken("", lib.GetRootVaultUrl())
	cl.ClearToken()
	secret, err := cl.Logical().Write(lib.IamAuthPluginPath+"/login", map[string]interface{}{
		"method":                          "sapassword",
		"service_account_password_uuid":   password.UUID,
		"service_account_password_secret": password.Secret,
		"roles":                           roles,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(secret).ToNot(BeNil())
	Expect(secret.Auth).ToNot(BeNil())
	cl.SetToken(secret.Auth.ClientToken)
	return pkg.VaultClient{Client: cl}
}

func (st Suite) createRoleIfNotExist(roleName string, scope model.RoleScope) {
	if scope == "" {
		scope = model.RoleScopeProject
	}
	roleAPI := lib.NewRoleAPI(st.IamVaultClient)
	var roleNotExists bool
	rawRole := roleAPI.Read(tools.Params{
		"name":         roleName,
		"expectStatus": tests.ExpectStatus("%d > 0"),
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

func (st Suite) WaitPrepareForSSHTesting(cfg CheckingEnvironment, maxAttempts int) error {
	f := func() error {
		return lib.TryLoginByMultipassJWTToAuthVault(cfg.UserMultipassJWT, lib.GetAuthVaultUrl())
	}
	return lib.Repeat(f, maxAttempts)
}

func (st Suite) WaitPrepareForLoginTesting(cfg CheckingEnvironment, maxAttempts int) error {
	_, multipassJWT := specs.CreateUserMultipass(lib.NewUserMultipassAPI(st.IamVaultClient),
		cfg.User, "test", 100*time.Second, 1000*time.Second, []string{"ssh"})
	f := func() error { return lib.TryLoginByMultipassJWTToAuthVault(multipassJWT, lib.GetAuthVaultUrl()) }
	return lib.Repeat(f, maxAttempts)
}

type CheckingEnvironmentTeammate struct {
	FlantTenant model.Tenant
	Admin       model.User
}

func (st *Suite) PrepareForTeammateGotSSHAccess() CheckingEnvironmentTeammate {
	var result CheckingEnvironmentTeammate
	// create flant tenant
	result.FlantTenant = model.Tenant{
		ArchiveMark: memdb.ArchiveMark{},
		UUID:        FlantTenantUUID,
		Identifier:  FlantTenantID,
	}

	// create a role 'iam_auth_read' if not exists
	st.createRoleIfNotExist(IamAuthRead, model.RoleScopeTenant)

	// create some user at the tenant
	result.Admin = specs.CreateRandomUser(lib.NewUserAPI(st.IamVaultClient), result.FlantTenant.UUID)
	fmt.Printf("Created admin:%#v\n", result.Admin)
	return result
}
