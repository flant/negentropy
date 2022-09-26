package server_accessd_init

import (
	_ "embed"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	tsc "github.com/flant/negentropy/e2e/tests/lib/test_server_and_client_preparing"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

var s tsc.Suite

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
		err := tsc.RunAndCheckAuthdAtServer(s, "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("check run server_accessd", func() {
		err := tsc.RunAndCheckServerAccessd(s, cfg.User.Identifier, cfg.TestServer.UUID, cfg.User.UUID)
		Expect(err).ToNot(HaveOccurred())
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
