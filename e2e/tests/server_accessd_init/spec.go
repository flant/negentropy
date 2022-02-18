package server_accessd_init

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
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
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

var s test_server_and_client_preparing.Suite

var flantIamSuite flant_iam_preparing.Suite

var cfg flant_iam_preparing.CheckingEnvironment

var _ = BeforeSuite(func() {
	s.BeforeSuite()
	flantIamSuite.BeforeSuite()
	Describe("configuring system", func() {
		cfg = flantIamSuite.PrepareForSSHTesting()
		err := flantIamSuite.WaitPrepareForSSHTesting(cfg, 40)
		Expect(err).ToNot(HaveOccurred())
	})
}, 10.0)

var _ = Describe("Process of server initializing by using server_accessd init", func() {
	cfgPath := "/opt/server-access/config.yaml"
	cfgFolder := "/opt/server-access"
	jwtPath := "/opt/authd/server-jwt"
	jwtFolder := "/opt/authd"

	It("prepare: check existence some files, and absence others", func() {
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
		tenantID := cfg.Tenant.Identifier
		projectID := cfg.Project.Identifier
		initCmd := fmt.Sprintf(s.ServerAccessdPath+" init --vault_url %s ", os.Getenv("ROOT_VAULT_INTERNAL_URL"))
		initCmd += fmt.Sprintf("-t %s -p %s ", tenantID, projectID)
		encodedPassword := b64.StdEncoding.EncodeToString([]byte(cfg.ServiceAccountPassword.UUID + ":" + cfg.ServiceAccountPassword.Secret))
		initCmd += fmt.Sprintf("--password %s ", encodedPassword)
		initCmd += fmt.Sprintf("--identifier %s ", flant_iam_preparing.TestServerIdentifier+"_new_ID")
		initCmd += "-l system=ubuntu "
		initCmd += fmt.Sprintf("--connection_hostname %s ", flant_iam_preparing.TestServerIdentifier)
		initCmd += fmt.Sprintf("--store_multipass_to %s ", jwtPath)
		s.ExecuteCommandAtContainer(s.TestServerContainer, []string{"/bin/bash", "-c", initCmd}, nil)
		lib.WaitDataReachFlantAuthPlugin(40, lib.GetAuthVaultUrl())
	})

	It("check files are created, and got server registration info", func() {
		// check jwt file exists
		Expect(s.CheckFileExistAtContainer(s.TestServerContainer, jwtPath, "f")).
			ToNot(HaveOccurred(), "jwt should exists")
		// clear token
		cfg.TestServerServiceAccountMultipassJWT = ""

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
		cfg.TestServer = readServerFromIam(cfg.Tenant.UUID, cfg.Project.UUID, serverUUID)
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
		time.Sleep(time.Second)
		s.RunDaemonAtContainer(s.TestServerContainer, s.AuthdPath, "server_authd.log")
		pidAuthd := s.FirstProcessPIDAtContainer(s.TestServerContainer, s.AuthdPath)
		Expect(pidAuthd).Should(BeNumerically(">", 0), "pid greater 0")
	})

	It("check run server_accessd", func() {
		// TODO check content /etc/nsswitch.conf
		err := s.CreateIfNotExistsDirectoryAtContainer(s.TestServerContainer, "/opt/serveraccessd")
		Expect(err).ToNot(HaveOccurred(), "folder should be created")
		s.KillAllInstancesOfProcessAtContainer(s.TestServerContainer, s.ServerAccessdPath)
		s.RunDaemonAtContainer(s.TestServerContainer, s.ServerAccessdPath, "server_accessd.log")
		pidServerAccessd := s.FirstProcessPIDAtContainer(s.TestServerContainer, s.ServerAccessdPath)
		Expect(pidServerAccessd).Should(BeNumerically(">", 0), "pid greater 0")
		time.Sleep(time.Second)
		authKeysFilePath := filepath.Join("/home", cfg.User.Identifier, ".ssh", "authorized_keys")
		contentAuthKeysFile := s.ExecuteCommandAtContainer(s.TestServerContainer,
			[]string{"/bin/bash", "-c", "cat " + authKeysFilePath}, nil)
		Expect(contentAuthKeysFile).To(HaveLen(1), "cat authorize should have one line text")
		principal := calculatePrincipal(cfg.TestServer.UUID, cfg.User.UUID)
		Expect(contentAuthKeysFile[0]).To(MatchRegexp(".+cert-authority,principals=\""+principal+"\" ssh-rsa.{373}"),
			"content should be specific")
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
