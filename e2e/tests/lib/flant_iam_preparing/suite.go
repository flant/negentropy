package flant_iam_preparing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/cli/pkg"
	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var serverLabels = map[string]string{"system": "ubuntu20"}

type Suite struct {
	IamVaultClient *http.Client
	// IamAuthVaultClient *http.Client
}

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

func (st *Suite) PrepareForLoginTesting() CheckingEnvironment {
	const (
		registerServerRole = "register_server"
		iam_auth_read      = "iam_auth_read"
	)
	var result CheckingEnvironment
	// create some tenant
	result.Tenant = specs.CreateRandomTenant(lib.NewTenantAPI(st.IamVaultClient))
	fmt.Printf("Created tenant:%#v\n", result.Tenant)
	// create some SA at the tenant for server registration
	result.ServiceAccount = specs.CreateRandomServiceAccount(lib.NewServiceAccountAPI(st.IamVaultClient), result.Tenant.UUID)
	fmt.Printf("Created serviceAccount:%#v\n", result.ServiceAccount)
	// create  SA password for SA login
	result.ServiceAccountPassword = specs.CreateServiceAccountPassword(lib.NewServiceAccountPasswordAPI(st.IamVaultClient),
		result.ServiceAccount, "test", 100*time.Second, []string{"register_server", "iam_read"})
	fmt.Printf("Created serviceAccountPassword:%#v\n", result.ServiceAccountPassword)

	// create a role 'register_server' if not exists
	st.createRoleIfNotExist(registerServerRole)

	// create a role 'iam_auth_read' if not exists
	st.createRoleIfNotExist(iam_auth_read)

	//time.Sleep(time.Second * 15)

	// create rolebinding for a sa in project with the registerServerRole
	result.ServiceAccountRoleBinding = specs.CreateRoleBinding(lib.NewRoleBindingAPI(st.IamVaultClient),
		model.RoleBinding{
			TenantUUID: result.Tenant.UUID,
			Version:    "",
			Identifier: uuid.New(),
			ValidTill:  1000000,
			RequireMFA: false,
			Members: []model.MemberNotation{{
				Type: model.ServiceAccountType,
				UUID: result.ServiceAccount.UUID,
			}},
			AnyProject: true,
			Roles: []model.BoundRole{{Name: registerServerRole, Options: map[string]interface{}{}},
				{Name: iam_auth_read, Options: map[string]interface{}{}}},
		})
	fmt.Printf("Created rolebinding:%#v\n", result.ServiceAccountRoleBinding)

	// create some project
	result.Project = specs.CreateRandomProject(lib.NewProjectAPI(st.IamVaultClient), result.Tenant.UUID)
	fmt.Printf("Created project:%#v\n", result.Project)
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

	err := st.WaitPrepareForLoginTesting(result, 25)
	Expect(err).ToNot(HaveOccurred())

	// create a role 'ssh' if not exists
	st.createRoleIfNotExist(sshRole)

	// create a role 'servers' if not exists
	st.createRoleIfNotExist(serverRole)

	// create a group with the user
	result.Group = specs.CreateRandomGroupWithUser(lib.NewGroupAPI(st.IamVaultClient), result.User.TenantUUID, result.User.UUID)
	fmt.Printf("Created group:%#v\n", result.Group)

	// create a role 'ssh' if not exists
	st.createRoleIfNotExist(sshRole)

	// create a role 'servers' if not exists
	st.createRoleIfNotExist(serverRole)

	// create rolebinding for a user in project with the ssh role
	result.UserRolebinding = specs.CreateRoleBinding(lib.NewRoleBindingAPI(st.IamVaultClient),
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

func (st Suite) WaitPrepareForSSHTesting(cfg CheckingEnvironment, maxAttempts int) error {
	f := func() error { return tryLoginByMultipassJWTToAuthVault(cfg.UserMultipassJWT) }
	return repeat(f, maxAttempts)
}

func (st Suite) WaitPrepareForLoginTesting(cfg CheckingEnvironment, maxAttempts int) error {
	_, multipassJWT := specs.CreateUserMultipass(lib.NewUserMultipassAPI(st.IamVaultClient),
		cfg.User, "test", 100*time.Second, 1000*time.Second, []string{"ssh"})
	f := func() error { return tryLoginByMultipassJWTToAuthVault(multipassJWT) }
	return repeat(f, maxAttempts)
}

func repeat(f func() error, maxAttempts int) error {
	err := f()
	counter := 1
	for err != nil {
		if counter > maxAttempts {
			return fmt.Errorf("exceeded attempts, last err:%w", err)
		}
		fmt.Printf("waiting fail %d attempt\n", counter)
		time.Sleep(time.Second)
		counter++
		err = f()
	}
	fmt.Printf("waiting completed successfully, attempt %d\n", counter)
	return nil
}

func tryLoginByMultipassJWTToAuthVault(multipassJWT string) error {
	vaultUrl := lib.GetAuthVaultUrl()
	url := vaultUrl + "/v1/auth/flant_iam_auth/login"
	payload := map[string]interface{}{
		"method": "multipass",
		"jwt":    multipassJWT,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil
	}
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("wrong response status:%d", resp.StatusCode)
	}
	return nil
}
