package teammate_got_ssh_access

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/e2e/tests/lib/test_server_and_client_preparing"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

var s test_server_and_client_preparing.Suite

var flantIamSuite flant_iam_preparing.Suite

var cfg flant_iam_preparing.CheckingEnvironmentTeammate

var _ = BeforeSuite(func() {
	s.BeforeSuite()
	flantIamSuite.BeforeSuite()
	Describe("configuring system", func() {
		cfg = flantIamSuite.PrepareForTeammateGotSSHAccess()

		err := flantIamSuite.WaitPrepareForTeammateGotSSHAccess(cfg, 40)
		Expect(err).ToNot(HaveOccurred())
		s.CheckServerBinariesAndFoldersExists()
		s.CheckClientBinariesAndFoldersExists()
	})
}, 1.0)

var _ = Describe("Process of getting ssh access to server by a teammate", func() {
	var adminClient *http.Client

	It("Configuring flant_flow, using Admin account", func() {
		// login c oidc
		adminAccessToken, err := tools.GetOIDCAccessToken(cfg.Admin.UUID)
		Expect(err).ToNot(HaveOccurred())

		adminVST := tools.LoginAccessToken(true, map[string]interface{}{
			"method": "okta-jwt", "jwt": adminAccessToken,
			"roles": []map[string]interface{}{
				{"role": "flow_write"}, {"role": "iam_write_all"},
			},
		}, lib.GetRootVaultUrl()).ClientToken
		adminClient = lib.NewIamVaultClient(adminVST)
	})

	var client ext_model.Client
	var saRegisterServerPassword model.ServiceAccountPassword
	var devopsTeam ext_model.Team
	var teammate ext_model.FullTeammate

	It("Interact flant_flow and flant_iam, using Admin client", func() {
		client = specs.CreateRandomClient(lib.NewFlowClientAPI(adminClient))
		saRegisterServer := iam_specs.CreateRandomServiceAccount(lib.NewServiceAccountAPI(adminClient), client.UUID)
		iam_specs.CreateRoleBinding(lib.NewRoleBindingAPI(adminClient),
			model.RoleBinding{
				TenantUUID: client.UUID,
				Identifier: uuid.New(),
				ValidTill:  1000000,
				RequireMFA: false,
				Members: []model.MemberNotation{{
					Type: model.ServiceAccountType,
					UUID: saRegisterServer.UUID,
				}},
				AnyProject: true,
				Roles: []model.BoundRole{{Name: flant_iam_preparing.RegisterServerRole, Options: map[string]interface{}{}},
					{Name: flant_iam_preparing.IamAuthRead, Options: map[string]interface{}{}}},
			})

		saRegisterServerPassword = iam_specs.CreateServiceAccountPassword(lib.NewServiceAccountPasswordAPI(adminClient),
			saRegisterServer, "server_register", 100*time.Second, []string{flant_iam_preparing.RegisterServerRole, flant_iam_preparing.IamReadRole})

		devopsTeam = specs.CreateDevopsTeam(lib.NewFlowTeamAPI(adminClient))
		teammate = specs.CreateRandomTeammate(lib.NewFlowTeammateAPI(adminClient), devopsTeam)

	})

	var project *ext_model.Project

	It("Create project with devops service_pack, using admin client", func() {
		projectAPI := lib.NewFlowProjectAPI(adminClient)
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
		initCmd := s.ServerAccessdPath + " init --vault_url http://vault-root:8200 "
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
		// Authd can be configured and run at Test_server
		err := s.CreateIfNotExistsDirectoryAtContainer(s.TestServerContainer,
			"/etc/flant/negentropy/authd-conf.d")
		Expect(err).ToNot(HaveOccurred(), "folder should be created")

		t, err := template.New("").Parse(test_server_and_client_preparing.ServerMainCFGTPL)
		Expect(err).ToNot(HaveOccurred(), "template should be ok")
		var serverMainCFG bytes.Buffer
		err = t.Execute(&serverMainCFG, s)
		Expect(err).ToNot(HaveOccurred(), "template should be executed")

		err = s.WriteFileToContainer(s.TestServerContainer,
			"/etc/flant/negentropy/authd-conf.d/main.yaml", serverMainCFG.String())
		Expect(err).ToNot(HaveOccurred(), "file should be written")

		err = s.WriteFileToContainer(s.TestServerContainer,
			"/etc/flant/negentropy/authd-conf.d/sock1.yaml", test_server_and_client_preparing.ServerSocketCFG)
		Expect(err).ToNot(HaveOccurred(), "file should be written")

		s.KillAllInstancesOfProcessAtContainer(s.TestServerContainer, s.AuthdPath)
		s.RunDaemonAtContainer(s.TestServerContainer, s.AuthdPath, "server_authd.log")
		time.Sleep(time.Second)
		pidAuthd := s.FirstProcessPIDAtContainer(s.TestServerContainer, s.AuthdPath)
		Expect(pidAuthd).Should(BeNumerically(">", 0), "pid greater 0")
	})

	It("check run server_accessd", func() {
		// TODO check content /etc/nsswitch.conf
		err := s.CreateIfNotExistsDirectoryAtContainer(s.TestServerContainer, "/opt/serveraccessd")
		Expect(err).ToNot(HaveOccurred(), "folder should be created")

		s.KillAllInstancesOfProcessAtContainer(s.TestServerContainer, s.ServerAccessdPath)
		s.RunDaemonAtContainer(s.TestServerContainer, s.ServerAccessdPath, "server_accessd.log")
		time.Sleep(time.Second)
		pidServerAccessd := s.FirstProcessPIDAtContainer(s.TestServerContainer, s.ServerAccessdPath)
		Expect(pidServerAccessd).Should(BeNumerically(">", 0), "pid greater 0")

		authKeysFilePath := filepath.Join("/home", cfg.FlantTenant.Identifier, teammate.Identifier, ".ssh", "authorized_keys")
		contentAuthKeysFile := s.ExecuteCommandAtContainer(s.TestServerContainer,
			[]string{"/bin/bash", "-c", "cat " + authKeysFilePath}, nil)
		Expect(contentAuthKeysFile).To(HaveLen(1), "cat authorize should have one line text")
		principal := calculatePrincipal(testServer.UUID, teammate.UUID)
		Expect(contentAuthKeysFile[0]).To(MatchRegexp(".+cert-authority,principals=\""+principal+"\" ssh-rsa.{373}"),
			"content should be specific")
	})

	var teammateClient *http.Client

	It("Teammate login negentropy", func() {
		teammateAccessToken, err := tools.GetOIDCAccessToken(teammate.UUID)
		Expect(err).ToNot(HaveOccurred())

		teammateVST := tools.LoginAccessToken(true, map[string]interface{}{
			"method": "okta-jwt", "jwt": teammateAccessToken,
			"roles": []map[string]interface{}{
				{"role": "iam_write", "tenant_uuid": cfg.FlantTenant.UUID},
			},
		}, lib.GetRootVaultUrl()).ClientToken

		teammateClient = lib.NewIamVaultClient(teammateVST)
	})

	var multipassJWT string

	It("Teammate create multipass", func() {
		_, multipassJWT = iam_specs.CreateUserMultipass(lib.NewUserMultipassAPI(teammateClient),
			teammate.User, "test", 100*time.Second, 1000*time.Second, []string{"ssh"})
		fmt.Printf("%#v\n", *teammateClient)
	})

	It("configuring client, using multipass", func() {
		s.PrepareClientForSSHTesting(flant_iam_preparing.CheckingEnvironment{UserMultipassJWT: multipassJWT})
	})

	It("cli ssh command works", func() {
		cfg := flant_iam_preparing.CheckingEnvironment{
			Tenant:     client,
			Project:    project.Project,
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

		Expect(s.DirectoryAtContainerNotExistOrEmpty(s.TestClientContainer, "/tmp/flint")).To(BeTrue(),
			"/tmp/flint files doesn't exist after closing cli")

		Expect(s.CheckFileExistAtContainer(s.TestServerContainer, testFilePath, "f")).
			ToNot(HaveOccurred(), "after run cli ssh - test file is created at server")

		Expect(s.CheckFileExistAtContainer(s.TestClientContainer, "/tmp/flint", "s")).
			ToNot(HaveOccurred(), "after run cli ssh - tmp dir exists at client container")

		Expect(s.DirectoryAtContainerNotExistOrEmpty(s.TestClientContainer, "/tmp/flint")).To(BeTrue(),
			"/tmp/flint is empty  after closing cli")
	})
})

func readServerFromIam(tenantUUID model.TenantUUID, projectUUID model.ProjectUUID, serverUUID ext.ServerUUID) ext.Server {
	cl, err := api.NewClient(api.DefaultConfig())
	Expect(err).ToNot(HaveOccurred())
	cl.SetToken(lib.GetRootRootToken())
	cl.SetAddress(lib.GetRootVaultUrl())
	secret, err := cl.Logical().Read(fmt.Sprintf("/flant_iam/tenant/%s/project/%s/server/%s", tenantUUID, projectUUID, serverUUID))
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

func calculatePrincipal(serverUUID string, userUUID model.UserUUID) string {
	principalHash := sha256.New()
	principalHash.Write([]byte(serverUUID))
	principalHash.Write([]byte(userUUID))
	principalSum := principalHash.Sum(nil)
	return fmt.Sprintf("%x", principalSum)
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

//flant/user_identifier
