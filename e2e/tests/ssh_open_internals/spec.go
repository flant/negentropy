package ssh_access_internals

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	_ "embed"
	"fmt"

	vault_api "github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
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
	It("direct sign certificate 30 times", func() {
		for i := 0; i < 30; i++ {
			var secret *vault_api.SecretAuth
			secret = userDirectLoginForSignCert(cfg.TestServer.TenantUUID, cfg.TestServer.ProjectUUID, cfg.TestServer.UUID)

			privateRSA, err := rsa.GenerateKey(rand.Reader, 2048)
			Expect(err).ToNot(HaveOccurred())

			pubkey, err := ssh.NewPublicKey(&privateRSA.PublicKey)
			Expect(err).ToNot(HaveOccurred())

			hash := sha256.Sum256([]byte(cfg.TestServer.UUID + cfg.User.UUID))
			principal := fmt.Sprintf("%x", hash)

			data := map[string]interface{}{
				"public_key":       string(ssh.MarshalAuthorizedKey(pubkey)),
				"valid_principals": principal,
			}

			cl := configure.GetClientWithToken("", lib.GetAuthVaultUrl())
			cl.ClearToken()
			cl.SetToken(secret.ClientToken)

			ssh := cl.SSHWithMountPoint("ssh")
			s, err := ssh.SignKey("signer", data)
			Expect(err).ToNot(HaveOccurred())
			fmt.Printf("\n======================\n")
			fmt.Printf("%#v\n", s)
		}
	})
})

const SSHOpenRole = "ssh.open"

func userDirectLoginForSignCert(tenantUUID, projectUUID, serverUUID string) *vault_api.SecretAuth {
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
					"servers": []string{serverUUID},
				},
			},
		}}

	cl := configure.GetClientWithToken("", lib.GetAuthVaultUrl())
	cl.ClearToken()

	secret, err := cl.Logical().Write(lib.IamAuthPluginPath+"/login", params)
	Expect(err).ToNot(HaveOccurred())
	Expect(secret).ToNot(BeNil())
	Expect(secret.Auth).ToNot(BeNil())
	return secret.Auth
}
