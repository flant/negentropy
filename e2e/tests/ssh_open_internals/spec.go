package ssh_access_internals

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"sort"
	"strings"

	vault_api "github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"

	"github.com/flant/negentropy/authd/pkg/daemon"
	"github.com/flant/negentropy/cli/pkg"
	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	tsc "github.com/flant/negentropy/e2e/tests/lib/test_server_and_client_preparing"
)

var flantIamSuite flant_iam_preparing.Suite

var cfg flant_iam_preparing.CheckingEnvironment

var _ = BeforeSuite(func() {
	flantIamSuite.BeforeSuite()
	Describe("configuring system", func() {
		cfg = flantIamSuite.PrepareForSSHTesting()
		err := flantIamSuite.WaitPrepareForSSHTesting(cfg, 40)
		Expect(err).ToNot(HaveOccurred())
	})
}, 1.0)

var _ = Describe("Process of getting ssh access to server by a cli", func() {
	It("direct sign certificate 10 times", func() {
		for i := 0; i < 10; i++ {
			pubkey, principals, serversUUIDs := pubkeyAndPrincipals(i)

			var secret *vault_api.SecretAuth
			secret = userDirectLoginForSignCert(lib.GetAuthVaultUrl(), cfg.TestServer.TenantUUID, cfg.TestServer.ProjectUUID, serversUUIDs)

			data := map[string]interface{}{
				"public_key":       string(ssh.MarshalAuthorizedKey(pubkey)),
				"valid_principals": principals,
			}

			cl := configure.GetClientWithToken("", lib.GetAuthVaultUrl())
			cl.ClearToken()
			cl.SetToken(secret.ClientToken)

			ssh := cl.SSHWithMountPoint("ssh")
			_, err := ssh.SignKey("signer", data)

			Expect(err).ToNot(HaveOccurred())
			fmt.Printf("principals: %s\n", principals)
		}
	})

	It("sign certificate 10 times through authd", func() {
		stopChan := runAuthdDaemonAtLocalhost()
		defer func() {
			stopChan <- struct{}{}
		}()
		curFolder, err := os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		authdSocketPath := curFolder + "/" + "authd.sock"
		err = os.Setenv("AUTHD_SOCKET_PATH", authdSocketPath)
		Expect(err).ToNot(HaveOccurred())
		for i := 0; i < 10; i++ {

			cliVaultClient, err := pkg.DefaultVaultClient()
			Expect(err).ToNot(HaveOccurred())

			pubkey, principals, serversUUIDs := pubkeyAndPrincipals(i)

			sshRequest := pkg.VaultSSHSignRequest{
				PublicKey:       string(ssh.MarshalAuthorizedKey(pubkey)),
				ValidPrincipals: principals,
			}

			_, err = cliVaultClient.SignPublicSSHCertificate(cfg.TestServer.TenantUUID, cfg.TestServer.ProjectUUID, serversUUIDs, sshRequest)

			Expect(err).ToNot(HaveOccurred())
			fmt.Printf("principals: %s\n", principals)
		}
	})
})

// runAuthdDaemonAtLocalhost returns chan for stopping
func runAuthdDaemonAtLocalhost() chan struct{} {
	curFolder, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile("client-jwt", []byte(cfg.UserMultipassJWT), 0600)
	Expect(err).ToNot(HaveOccurred())
	err = os.Setenv("VAULT_CACERT", "../../../docker/vault/tls/ca.crt")
	Expect(err).ToNot(HaveOccurred())

	clientMainCfg := tsc.ClientMainAuthdCFG(&tsc.MainAuthdCfgV1{
		DefaultSocketDirectory: curFolder,
		JwtPath:                curFolder + "/" + "client-jwt",
		RootVaultInternalURL:   lib.GetRootVaultUrl(),
		AuthVaultInternalURL:   lib.GetAuthVaultUrl(),
	})
	err = os.WriteFile("main.yaml", []byte(clientMainCfg), 0666)
	Expect(err).ToNot(HaveOccurred())

	clientSocketCfg := tsc.ClientSocketCFG
	err = os.WriteFile("sock1.yaml", []byte(clientSocketCfg), 0777)
	Expect(err).ToNot(HaveOccurred())

	authd := daemon.NewDefaultAuthd()
	authd.Config.ConfDirectory = curFolder
	err = authd.Start()
	Expect(err).ToNot(HaveOccurred())

	stopChan := make(chan struct{})
	go func() {
		<-stopChan
		authd.Stop()
	}()
	return stopChan
}

const SSHOpenRole = "ssh.open"

func userDirectLoginForSignCert(vaultApiUrl, tenantUUID, projectUUID string, serverUUIDs []string) *vault_api.SecretAuth {
	params := map[string]interface{}{
		"method": "multipass",
		"jwt":    cfg.UserMultipassJWT,
		"roles": []interface{}{
			map[string]interface{}{
				"role":         SSHOpenRole,
				"tenant_uuid":  tenantUUID,
				"project_uuid": projectUUID,
				"claim": map[string]interface{}{
					"ttl":     "720m",
					"max_ttl": "1440m",
					"servers": serverUUIDs,
				},
			},
		}}

	cl := configure.GetClientWithToken("", vaultApiUrl)

	secret, err := cl.Logical().Write(lib.IamAuthPluginPath+"/login", params)
	Expect(err).ToNot(HaveOccurred())
	Expect(secret).ToNot(BeNil())
	Expect(secret.Auth).ToNot(BeNil())
	return secret.Auth
}

// testServers provides different cases for testing
func testServers(i int) []string {
	set := [][]string{{cfg.TestServer.UUID}, {cfg.TestServer2.UUID}, {cfg.TestServer.UUID, cfg.TestServer2.UUID}, {cfg.TestServer2.UUID, cfg.TestServer.UUID}}
	return set[i%4]
}

func pubkeyAndPrincipals(i int) (ssh.PublicKey, string, []string) {
	privateRSA, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).ToNot(HaveOccurred())

	pubkey, err := ssh.NewPublicKey(&privateRSA.PublicKey)
	Expect(err).ToNot(HaveOccurred())

	serverUUIDs := testServers(i)
	principals := []string{}
	for _, serverUUID := range serverUUIDs {
		hash := sha256.Sum256([]byte(serverUUID + cfg.User.UUID))
		principals = append(principals, fmt.Sprintf("%x", hash))
	}
	sort.Strings(principals)

	return pubkey, strings.Join(principals, ","), serverUUIDs
}
