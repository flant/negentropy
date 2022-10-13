package teammate_got_ssh_access

import (
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	tsc "github.com/flant/negentropy/e2e/tests/lib/test_server_and_client_preparing"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

var s tsc.Suite

var flantIamSuite flant_iam_preparing.Suite

var cfg flant_iam_preparing.CheckingEnvironmentTeammate

var _ = BeforeSuite(func() {
	s.BeforeSuite()
	flantIamSuite.BeforeSuite()
	Describe("configuring system", func() {
		cfg = flantIamSuite.PrepareForTeammateGotSSHAccess()

		err := lib.WaitDataReachFlantAuthPlugin(40, lib.GetAuthVaultUrl())
		Expect(err).ToNot(HaveOccurred())
		s.CheckServerBinariesAndFoldersExists()
		s.CheckClientBinariesAndFoldersExists()
	})
}, 1.0)

var _ = Describe("Process of getting ssh access to server by a teammate", func() {
	var adminClient *http.Client

	It("Configuring flant_flow, using Admin account", func() {
		// login c oidc
		adminAccessToken, err := tools.GetOIDCAccessToken(cfg.Admin.UUID, cfg.Admin.Email)
		Expect(err).ToNot(HaveOccurred())

		adminVST := tools.LoginAccessToken(true, map[string]interface{}{
			"method": "okta-jwt", "jwt": adminAccessToken,
			"roles": []map[string]interface{}{
				{"role": flant_iam_preparing.FlantAdminRole},
				{"role": flant_iam_preparing.FlantClientManageRole, "tenant_uuid": flant_iam_preparing.FlantTenantUUID},
			},
		}, lib.GetRootVaultUrl()).ClientToken
		adminClient = lib.NewIamVaultClient(adminVST)
	})

	var client ext_model.Client
	var saRegisterServerPassword model.ServiceAccountPassword
	var devopsTeam ext_model.Team
	var teammate ext_model.FullTeammate
	var primaryAdminClient *http.Client

	It("Interact flant_flow and flant_iam, using Admin client & clientPrimaryAdmin client", func() {
		clientPrimaryAdmin := iam_specs.CreateRandomUser(lib.NewUserAPI(adminClient), cfg.FlantTenant.UUID)

		client = specs.CreateRandomClient(lib.NewFlowClientAPI(adminClient), clientPrimaryAdmin.UUID)

		time.Sleep(time.Second * 2)

		// login by primaryAdmin
		primaryAdminAccessToken, err := tools.GetOIDCAccessToken(clientPrimaryAdmin.UUID, clientPrimaryAdmin.Email)
		Expect(err).ToNot(HaveOccurred())

		prAdminVST := tools.LoginAccessToken(true, map[string]interface{}{
			"method": "okta-jwt", "jwt": primaryAdminAccessToken,
			"roles": []map[string]interface{}{
				{"role": flant_iam_preparing.FlantClientManageRole, "tenant_uuid": client.UUID},
			},
		}, lib.GetRootVaultUrl()).ClientToken
		primaryAdminClient = lib.NewIamVaultClient(prAdminVST)

		saRegisterServer := iam_specs.CreateRandomServiceAccount(lib.NewServiceAccountAPI(primaryAdminClient), client.UUID)
		iam_specs.CreateRoleBinding(lib.NewRoleBindingAPI(primaryAdminClient),
			model.RoleBinding{
				TenantUUID:  client.UUID,
				Description: "teammate got ssh access testing",
				ValidTill:   10_000_000_000,
				RequireMFA:  false,
				Members: []model.MemberNotation{{
					Type: model.ServiceAccountType,
					UUID: saRegisterServer.UUID,
				}},
				AnyProject: true,
				Roles:      []model.BoundRole{{Name: flant_iam_preparing.RegisterServerRole, Options: map[string]interface{}{"max_ttl": "1600m", "ttl": "800m"}}},
			})

		saRegisterServerPassword = iam_specs.CreateServiceAccountPassword(lib.NewServiceAccountPasswordAPI(primaryAdminClient),
			saRegisterServer, "server_register", 100*time.Second, []string{flant_iam_preparing.RegisterServerRole})

		devopsTeam = specs.CreateDevopsTeam(lib.NewFlowTeamAPI(adminClient))
		teammate = specs.CreateRandomTeammate(lib.NewFlowTeammateAPI(adminClient), devopsTeam)
	})

	var project *ext_model.Project

	It("Create project with devops service_pack, using admin client", func() {
		projectAPI := lib.NewFlowProjectAPI(primaryAdminClient)
		createPayload := map[string]interface{}{
			"tenant_uuid":   client.UUID,
			"service_packs": []string{ext_model.DevOps},
			"identifier":    "PNAME",
			"devops_team":   devopsTeam.UUID,
		}
		var err error
		project, err = specs.СreateProject(projectAPI, client.UUID, createPayload, false)
		Expect(err).ToNot(HaveOccurred())
	})

	cfgPath := "/opt/server-access/config.yaml"
	cfgFolder := "/opt/server-access"
	jwtPath := "/opt/authd/server-jwt"
	jwtFolder := "/opt/authd"

	It("prepare Server: check existence some files, and absence others", func() {
		err := s.CheckFileExistAtContainer(s.TestServerContainer, s.AuthdPath, "f")
		Expect(err).ToNot(HaveOccurred(), "Test_server should have authd")

		err = s.CheckFileExistAtContainer(s.TestServerContainer, s.ServerAccessdPath, "f")
		Expect(err).ToNot(HaveOccurred(), "Test_server should have server-accessd")

		err = s.DeleteFileAtContainer(s.TestServerContainer, cfgPath)
		Expect(err).ToNot(HaveOccurred(), "delete cfg file if exists")

		err = s.CreateIfNotExistsDirectoryAtContainer(s.TestServerContainer, cfgFolder)
		Expect(err).ToNot(HaveOccurred(), "folder should be created")

		err = s.DeleteFileAtContainer(s.TestServerContainer, jwtPath)
		Expect(err).ToNot(HaveOccurred(), "delete jwt file if exists")

		err = s.CreateIfNotExistsDirectoryAtContainer(s.TestServerContainer, jwtFolder)
		Expect(err).ToNot(HaveOccurred(), "folder should be created")
	})

	It("run server_accessd init", func() {
		// Test_server server_accessd can be configured by server_accessd init
		// clear files which should be created by server_accessd init
		// run init
		tenantID := client.Identifier
		projectID := project.Identifier
		initCmd := fmt.Sprintf(s.ServerAccessdPath+" init --vault_url %s ", os.Getenv("ROOT_VAULT_INTERNAL_URL"))
		initCmd += fmt.Sprintf("-t %s -p %s ", tenantID, projectID)
		encodedPassword := b64.StdEncoding.EncodeToString([]byte(saRegisterServerPassword.UUID + ":" + saRegisterServerPassword.Secret))
		initCmd += fmt.Sprintf("--password %s ", encodedPassword)
		initCmd += fmt.Sprintf("--identifier %s ", "test-server")
		initCmd += "-l system=ubuntu "
		initCmd += fmt.Sprintf("--connection_hostname %s ", flant_iam_preparing.TestServerIdentifier)
		initCmd += fmt.Sprintf("--store_multipass_to %s ", jwtPath)
		s.ExecuteCommandAtContainer(s.TestServerContainer, []string{"/bin/bash", "-c", initCmd}, nil)
	})

	var testServer ext.Server

	It("check files are created, and got server registration info", func() {
		// check jwt file exists
		Expect(s.CheckFileExistAtContainer(s.TestServerContainer, jwtPath, "f")).
			ToNot(HaveOccurred(), "jwt should exists")
		// clear token

		// read config and fill Server
		Expect(s.CheckFileExistAtContainer(s.TestServerContainer, cfgPath, "f")).
			ToNot(HaveOccurred(), "cfg file should exists")
		contentCfgFile := s.ExecuteCommandAtContainer(s.TestServerContainer,
			[]string{"/bin/bash", "-c", "cat " + cfgPath}, nil)
		Expect(contentCfgFile).To(HaveLen(5), "cat authorize should have one line text")
		// parse server_uuid
		serverRawData := contentCfgFile[2] // expect: "server: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX\n"
		splitted := strings.Split(serverRawData, ":")
		Expect(splitted).To(HaveLen(2))
		Expect(splitted[0]).To(Equal("server"))
		serverUUID := strings.TrimSpace(splitted[1])
		//
		testServer = readServerFromIam(client.UUID, project.UUID, serverUUID)
	})

	It("configure authd", func() {
		err := tsc.RunAndCheckAuthdAtServer(s, "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("check run server_accessd", func() {
		posixUserName := filepath.Join(cfg.FlantTenant.Identifier, teammate.Identifier)
		err := tsc.RunAndCheckServerAccessd(s, posixUserName, testServer.UUID, teammate.UUID)
		Expect(err).ToNot(HaveOccurred())
	})

	var teammateClient *http.Client
	var teammateVST string

	It("Teammate login negentropy", func() {
		teammateAccessToken, err := tools.GetOIDCAccessToken(teammate.UUID, teammate.Email)
		Expect(err).ToNot(HaveOccurred())

		teammateVST = tools.LoginAccessToken(true, map[string]interface{}{
			"method": "okta-jwt", "jwt": teammateAccessToken,
			"roles": []map[string]interface{}{
				{"role": "flant.teammate"},
			},
		}, lib.GetRootVaultUrl()).ClientToken

		teammateClient = lib.NewIamVaultClient(teammateVST)
	})

	It("Teammate check effective roles", func() {
		cl := configure.GetClientWithToken(teammateVST, lib.GetRootVaultUrl())
		roles := []string{"flant.admin", "flant.client.manage", "servers.query", "ssh.open", "tenant.manage", "tenant.read", "tenant.read.auth", "tenants.list.auth"}
		payload := map[string]interface{}{"roles": roles}

		requestUrl := lib.IamAuthPluginPath + "/check_effective_roles"
		tsc.Try(5, func() error { // need here retries due to data kafka-flow delay
			secret, err := cl.Logical().Write(requestUrl, payload)
			Expect(err).ToNot(HaveOccurred())

			ers := secret.Data["effective_roles"].([]interface{})
			Expect(ers).To(HaveLen(len(roles)))
			tenantCounts := map[string]int{
				"servers.query":       1,
				"ssh.open":            1,
				"tenant.read":         1,
				"tenant.read.auth":    1,
				"flant.client.manage": 1,
				"tenant.manage":       1,
			}
			projectCounts := map[string]int{
				"servers.query":       1,
				"ssh.open":            1,
				"tenant.read.auth":    1,
				"flant.client.manage": 1,
				"tenant.manage":       1,
			}

			for i := range ers {
				role := roles[i]
				er := ers[i].(map[string]interface{})
				Expect(er["role"]).To(Equal(role))
				Expect(er).To(HaveKey("tenants"))
				tenants := er["tenants"].([]interface{})
				if len(tenants) != tenantCounts[role] {
					return fmt.Errorf("returns wrong amount of permitetd tenants: %d for role: %q, expected: %d",
						len(tenants), role, tenantCounts[role])
				}
				if tenantCounts[role] > 0 {
					tenant := tenants[0].(map[string]interface{})
					Expect(tenant).To(HaveKey("uuid"))
					Expect(tenant).To(HaveKey("identifier"))
					Expect(tenant).To(HaveKey("projects"))
					projects := tenant["projects"].([]interface{})
					Expect(len(projects)).To(Equal(projectCounts[role]))
					if len(projects) > 0 {
						project := projects[0].(map[string]interface{})
						Expect(project).To(HaveKey("uuid"))
						Expect(project).To(HaveKey("identifier"))
					}
				}
			}
			return nil
		})
	})

	var multipassJWT string

	It("Teammate create multipass", func() {
		_, multipassJWT = iam_specs.CreateUserMultipass(lib.NewUserMultipassAPI(teammateClient),
			teammate.User, "test", 100*time.Second, 1000*time.Second, []string{"ssh.open"})
		fmt.Printf("%#v\n", *teammateClient)
	})

	It("configuring client, using multipass", func() {
		s.PrepareClientForSSHTesting(flant_iam_preparing.CheckingEnvironment{UserMultipassJWT: multipassJWT})
	})

	It("cli ssh command works", func() {
		s.ExecuteCommandAtContainer(s.TestClientContainer, []string{
			"/bin/bash", "-c",
			"sudo rm -rf /tmp/flint",
		}, []string{})
		time.Sleep(time.Millisecond * 100)
		Expect(s.DirectoryAtContainerNotExistOrEmptyWithRetry(s.TestClientContainer, "/tmp/flint", 13)).ToNot(HaveOccurred(),
			"/tmp/flint files doesn't exist before start")

		cfg := flant_iam_preparing.CheckingEnvironment{
			Tenant: client,
			Project: model.Project{
				UUID:       project.UUID,
				TenantUUID: project.TenantUUID,
				Version:    project.Version,
				Identifier: project.Identifier,
			},
			User:       teammate.User,
			TestServer: testServer,
		}

		runningCliCmd := fmt.Sprintf("/opt/cli/bin/cli  ssh -t %s -p %s\n", cfg.Tenant.Identifier, cfg.Project.Identifier)
		sshCmd := fmt.Sprintf("ssh -oStrictHostKeyChecking=accept-new %s.%s", cfg.Project.Identifier,
			cfg.TestServer.Identifier)
		testFilePath := fmt.Sprintf("/home/flant/%s/test.txt", cfg.User.Identifier)
		touchCommand := "touch " + testFilePath
		fmt.Printf("\n%s\n", runningCliCmd)
		fmt.Printf("\n%s\n", sshCmd)
		fmt.Printf("\n%s\n", touchCommand)
		output := s.ExecuteCommandAtContainer(s.TestClientContainer, []string{
			"/bin/bash", "-c",
			runningCliCmd,
		},
			[]string{
				sshCmd,
				touchCommand,
				"exit", "exit",
			})

		writeLogToFile(output, fmt.Sprintf("cli.log"))

		Expect(s.DirectoryAtContainerNotExistOrEmptyWithRetry(s.TestClientContainer, "/tmp/flint/flant", 13)).ToNot(HaveOccurred(),
			"/tmp/flint/flant files doesn't exist after closing cli")

		Expect(s.CheckFileExistAtContainer(s.TestServerContainer, testFilePath, "f")).
			ToNot(HaveOccurred(), "after run cli ssh - test file is created at server")

		Expect(s.CheckFileExistAtContainer(s.TestClientContainer, "/tmp/flint", "s")).
			ToNot(HaveOccurred(), "after run cli ssh - tmp dir exists at client container")

		Expect(s.DirectoryAtContainerNotExistOrEmptyWithRetry(s.TestClientContainer, "/tmp/flint", 13)).ToNot(HaveOccurred(),
			"/tmp/flint is empty  after closing cli")
	})

	var devopsTeam2 ext_model.Team

	It("lost access after moving user to other team", func() {
		devopsTeam2 = specs.CreateDevopsTeam(lib.NewFlowTeamAPI(adminClient))
		checkChangeTeam := func(oldTeamUUID string, newTeamUUID string, usersLen int) {
			updatedData := lib.NewFlowTeammateAPI(adminClient).Update(tests.Params{
				"team":     oldTeamUUID,
				"teammate": teammate.UUID,
			}, nil, map[string]interface{}{
				"resource_version": teammate.Version,
				"role_at_team":     teammate.RoleAtTeam,
				"identifier":       teammate.Identifier,
				"new_team_uuid":    newTeamUUID,
				"email":            teammate.Email,
			})
			teammate.Version = updatedData.Get("teammate.resource_version").String()
			Expect(lib.WaitDataReachFlantAuthPlugin(40, lib.GetAuthVaultUrl())).ToNot(HaveOccurred())
			time.Sleep(time.Second * 2)
			users, err := getPosixUsers(lib.NewConfiguredIamAuthVaultClient(), client.UUID, project.UUID, testServer.UUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(users).To(HaveLen(usersLen))
		}
		checkChangeTeam(devopsTeam.UUID, devopsTeam2.UUID, 0)
		checkChangeTeam(devopsTeam2.UUID, devopsTeam.UUID, 1)
	})

	It("lost access after deleting teammate", func() {
		teammate2 := specs.CreateRandomTeammate(lib.NewFlowTeammateAPI(lib.NewConfiguredIamVaultClient()), devopsTeam)
		fmt.Printf("teammate1 %#v\n", teammate)
		fmt.Printf("teammate2 %#v\n", teammate2)
		Expect(lib.WaitDataReachFlantAuthPlugin(40, lib.GetAuthVaultUrl())).ToNot(HaveOccurred())
		time.Sleep(time.Second * 2)
		users, err := getPosixUsers(lib.NewConfiguredIamAuthVaultClient(), client.UUID, project.UUID, testServer.UUID)
		Expect(err).ToNot(HaveOccurred())
		Expect(users).To(HaveLen(2))

		lib.NewFlowTeammateAPI(adminClient).Delete(tests.Params{
			"team":     devopsTeam.UUID,
			"teammate": teammate2.UUID,
		}, nil)
		Expect(lib.WaitDataReachFlantAuthPlugin(40, lib.GetAuthVaultUrl())).ToNot(HaveOccurred())
		time.Sleep(time.Second * 2)
		users, err = getPosixUsers(lib.NewConfiguredIamAuthVaultClient(), client.UUID, project.UUID, testServer.UUID)
		Expect(err).ToNot(HaveOccurred())
		Expect(users).To(HaveLen(1))
	})

	It("lost access after changing devops team", func() {
		checkChangeDevopsTeam := func(newTeamUUID string, usersLen int) {
			updatedData := lib.NewFlowProjectAPI(primaryAdminClient).Update(tests.Params{
				"client":  client.UUID,
				"project": project.UUID,
			}, nil, map[string]interface{}{
				"devops_team":      newTeamUUID,
				"resource_version": project.Version,
				"service_packs":    []string{ext_model.DevOps},
				"identifier":       project.Identifier,
			})
			project.Version = updatedData.Get("project.resource_version").String()
			println("updatedData = " + updatedData.String())
			Expect(lib.WaitDataReachFlantAuthPlugin(40, lib.GetAuthVaultUrl())).ToNot(HaveOccurred())
			rbsListData := lib.NewRoleBindingAPI(primaryAdminClient).List(tests.Params{
				"tenant": client.UUID,
			}, map[string][]string{})
			println("rbsListData = " + rbsListData.String())
			users, err := getPosixUsers(lib.NewConfiguredIamAuthVaultClient(), client.UUID, project.UUID, testServer.UUID)
			fmt.Printf("%#v\n", users)
			Expect(err).ToNot(HaveOccurred())
			Expect(users).To(HaveLen(usersLen))
		}
		checkChangeDevopsTeam(devopsTeam2.UUID, 0)
		checkChangeDevopsTeam(devopsTeam.UUID, 1)
	})
})

func readServerFromIam(tenantUUID model.TenantUUID, projectUUID model.ProjectUUID, serverUUID ext.ServerUUID) ext.Server {
	cl := configure.GetClientWithToken(lib.GetRootRootToken(), lib.GetRootVaultUrl())
	secret, err := cl.Logical().Read(fmt.Sprintf("/flant/tenant/%s/project/%s/server/%s", tenantUUID, projectUUID, serverUUID))
	Expect(err).ToNot(HaveOccurred())
	Expect(secret).ToNot(BeNil())
	Expect(secret.Data).ToNot(BeNil())
	Expect(secret.Data).To(HaveKey("server"))
	serverData := secret.Data["server"]
	bytes, err := json.Marshal(serverData)
	Expect(err).ToNot(HaveOccurred())
	var server ext.Server
	err = json.Unmarshal(bytes, &server)
	Expect(err).ToNot(HaveOccurred())
	return server
}

func writeLogToFile(output []string, logFilePath string) {
	logFile, err := os.Create(logFilePath)
	if err != nil {
		panic(err)
	}
	for _, s := range output {
		logFile.WriteString(s)
	}
	logFile.Close()
}

// return user identifiers
func getPosixUsers(authClient *http.Client, tenantUUID string, projectUUID string, serverUUID string) ([]string, error) {
	url := fmt.Sprintf("/tenant/%s/project/%s/server/%s/posix_users",
		tenantUUID, projectUUID, serverUUID)
	method := "GET"
	fmt.Println(url)
	req, err := http.NewRequest(method, url, strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	res, err := authClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	json := tools.UnmarshalVaultResponse(data)

	Expect(json.Map()).To(HaveKey("posix_users"))
	jsonUsers := json.Get("posix_users")
	if jsonUsers.Type == gjson.Null {
		return []string{}, nil
	}
	Expect(jsonUsers.IsArray()).To(BeTrue())
	usersIdentifiers := []string{}
	for _, jsonUser := range jsonUsers.Array() {
		Expect(jsonUser.Map()).To(HaveKey("name"))
		usersIdentifiers = append(usersIdentifiers, jsonUser.Get("name").String())
	}
	return usersIdentifiers, nil
}
