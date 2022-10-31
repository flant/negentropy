package flant_gitops

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/vault"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault entities")
}

var _ = Describe("flant_gitops", func() {
	// vaults tls common certificate CA
	var vaultsCaCert string
	// at this vault will live flant_gitops in prod
	var confVault ConfVault
	// this vault will under flant_gitops upgrades
	var rootVault Vault
	// this is a repo watched by flant_gitops
	var testGitRepo *util.TestGitRepo
	// flantGitopsBackend to interact
	var b *TestableBackend

	BeforeSuite(func() {
		var err error
		vaultCaCrtData, err := ioutil.ReadFile("examples/conf/ca.crt") // it is common tls cert
		Expect(err).ToNot(HaveOccurred())
		vaultsCaCert = string(vaultCaCrtData)

		// run and configure conf vault
		confVault, err = StartAndConfigureConfVault()
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("%#v\n", confVault)

		// run and configure root vault
		rootVault, err = StartAndConfigureRootVault(confVault.caPEM)
		fmt.Printf("%#v\n", rootVault)

		// prepare  repo
		testGitRepo, err = util.NewTestGitRepo("flant_gitops_test_repo")
		Expect(err).ToNot(HaveOccurred())
		err = testGitRepo.WriteFileIntoRepoAndCommit("data", []byte("OUTPUT1\n"), "one")
		Expect(err).ToNot(HaveOccurred())

		// run testable flant_gitops_backend
		b, err = getTestBackend(context.Background())
		Expect(err).ToNot(HaveOccurred())
	}, 1.0)

	AfterSuite(func() {
		testGitRepo.Clean()
		confVault.Remove()
		rootVault.Remove()
	}, 1.0)

	Describe("flant_gitops configuring and running ", func() {
		It("configure self_access into conf vault", func() {
			req := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "configure_vault_access",
				Data: map[string]interface{}{
					"vault_addr":            confVault.Addr,
					"vault_tls_server_name": "vault_host",
					"role_name":             "good",
					"secret_id_ttl":         "360h",
					"approle_mount_point":   "auth/approle/",
					"secret_id":             confVault.secretID,
					"role_id":               confVault.roleID,
					"vault_cacert":          vaultsCaCert,
				},
				Storage:    b.Storage,
				Connection: &logical.Connection{},
			}
			resp, err := b.B.HandleRequest(context.Background(), req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp != nil && resp.IsError()).ToNot(BeTrue())
		})

		It("configure/git_repository, including first commit", func() {
			req := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "configure/git_repository",
				Data: map[string]interface{}{
					"git_repo_url":    testGitRepo.RepoDir,
					"git_branch_name": "master",
					"git_poll_period": "15m",
					"required_number_of_verified_signatures_on_commit": "0",
					"initial_last_successful_commit":                   testGitRepo.CommitHashes[0],
				},
				Storage:    b.Storage,
				Connection: &logical.Connection{},
			}
			resp, err := b.B.HandleRequest(context.Background(), req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp != nil && resp.IsError()).ToNot(BeTrue())
		})

		It("configure/vaults", func() {
			vaults := []map[string]interface{}{{
				"name":         rootVault.Name,
				"url":          rootVault.Addr,
				"vault_cacert": vaultsCaCert,
			}}

			req := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "configure/vaults",
				Data: map[string]interface{}{
					"vaults": vaults,
				},
				Storage:    b.Storage,
				Connection: &logical.Connection{},
			}
			resp, err := b.B.HandleRequest(context.Background(), req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp != nil && resp.IsError()).ToNot(BeTrue())
			printNewLogs(b.Logger)
		})

		Context("periodic function", func() {
			It("flant_gitops run periodic functions without any new commits", func() {
				ctx := context.Background()
				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectSavedWorkingCommits(ctx, b.Storage)
				Expect(err).ToNot(HaveOccurred())
				Expect(lastStartedCommit).To(Equal(""))
				Expect(lastPushedToK8sCommit).To(Equal(""))
				Expect(LastK8sFinishedCommit).To(Equal(""))

				err = b.B.PeriodicFunc(ctx, &logical.Request{Storage: b.Storage})
				Expect(err).ToNot(HaveOccurred())

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err = collectSavedWorkingCommits(ctx, b.Storage)
				Expect(err).ToNot(HaveOccurred())
				Expect(lastStartedCommit).To(Equal(""))
				Expect(lastPushedToK8sCommit).To(Equal(""))
				Expect(LastK8sFinishedCommit).To(Equal(""))
				Expect(b.MockKubeService.LenActiveJobs()).To(Equal(0))
				Expect(b.MockKubeService.LenFinishedJobs()).To(Equal(0))
				printNewLogs(b.Logger)
			})

			It("flant_gitops run periodic functions with new commit, did not exceed interval", func() {
				ctx := context.Background()
				err := updateLastRunTimeStamp(ctx, b.Storage, b.Clock.Now())
				Expect(err).ToNot(HaveOccurred())
				err = testGitRepo.WriteFileIntoRepoAndCommit("data", []byte("OUTPUT2\n"), "two")
				Expect(err).ToNot(HaveOccurred())

				err = b.B.PeriodicFunc(ctx, &logical.Request{Storage: b.Storage})
				Expect(err).ToNot(HaveOccurred())

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectSavedWorkingCommits(ctx, b.Storage)
				Expect(err).ToNot(HaveOccurred())
				Expect(lastStartedCommit).To(Equal(""))
				Expect(lastPushedToK8sCommit).To(Equal(""))
				Expect(LastK8sFinishedCommit).To(Equal(""))
				Expect(b.MockKubeService.LenActiveJobs()).To(Equal(0))
				Expect(b.MockKubeService.LenFinishedJobs()).To(Equal(0))
				printNewLogs(b.Logger)
			})

			It("flant_gitops run periodic functions with new commit and exceeding interval", func() {
				ctx := context.Background()
				b.Clock.SetNowTime(b.Clock.Now().Add(time.Minute * 16))

				err := b.B.PeriodicFunc(ctx, &logical.Request{Storage: b.Storage})
				Expect(err).ToNot(HaveOccurred())

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectSavedWorkingCommits(ctx, b.Storage)
				Expect(err).ToNot(HaveOccurred())
				err = tests.FastRepeat(func() error {
					if lastStartedCommit != testGitRepo.CommitHashes[1] { // change is here
						return fmt.Errorf("expected %q equals %q", lastStartedCommit, testGitRepo.CommitHashes[1])
					}
					return nil
				}, 50)
				Expect(err).ToNot(HaveOccurred())
				Expect(lastStartedCommit).To(Equal(testGitRepo.CommitHashes[1]))
				Expect(lastPushedToK8sCommit).To(Equal(""))
				Expect(LastK8sFinishedCommit).To(Equal(""))
				Expect(b.MockKubeService.LenFinishedJobs()).To(Equal(0))
				err = tests.FastRepeat(func() error {
					if b.MockKubeService.LenActiveJobs() != 1 { // change is here
						return fmt.Errorf("expected b.MockKubeService.LenActiveJobs() equals 1, got: %d", b.MockKubeService.LenActiveJobs())
					}
					return nil
				}, 50)
				Expect(err).ToNot(HaveOccurred())
				Expect(b.MockKubeService.HasActiveJob(testGitRepo.CommitHashes[1])).To(BeTrue()) // change is here
				printNewLogs(b.Logger)
			})

			It("flant_gitops run periodic functions after pushing  task to k8s", func() {
				ctx := context.Background()

				err := b.B.PeriodicFunc(ctx, &logical.Request{Storage: b.Storage})
				Expect(err).ToNot(HaveOccurred())

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectSavedWorkingCommits(ctx, b.Storage)
				Expect(err).ToNot(HaveOccurred())
				Expect(lastStartedCommit).To(Equal(testGitRepo.CommitHashes[1]))
				Expect(lastPushedToK8sCommit).To(Equal(testGitRepo.CommitHashes[1])) // change is here
				Expect(LastK8sFinishedCommit).To(Equal(""))
				Expect(b.MockKubeService.LenFinishedJobs()).To(Equal(0))
				printNewLogs(b.Logger)
			})

			It("flant_gitops run periodic functions after finishing task at k8s", func() {
				ctx := context.Background()
				err := b.MockKubeService.FinishJob(ctx, testGitRepo.CommitHashes[1])
				Expect(err).ToNot(HaveOccurred())

				err = b.B.PeriodicFunc(ctx, &logical.Request{Storage: b.Storage})
				Expect(err).ToNot(HaveOccurred())

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectSavedWorkingCommits(ctx, b.Storage)
				Expect(err).ToNot(HaveOccurred())
				Expect(lastStartedCommit).To(Equal(testGitRepo.CommitHashes[1]))
				Expect(lastPushedToK8sCommit).To(Equal(testGitRepo.CommitHashes[1]))
				Expect(LastK8sFinishedCommit).To(Equal(testGitRepo.CommitHashes[1])) // change is here
				Expect(b.MockKubeService.LenFinishedJobs()).To(Equal(1))             // change is here
				printNewLogs(b.Logger)
			})

			It("passed to k8s vaultsB64Json contains valid vaults creds", func() {
				job, err := b.MockKubeService.GetFinishedJob(testGitRepo.CommitHashes[1])
				Expect(err).ToNot(HaveOccurred())

				checkedVaults := parse(job.VaultsB64Json, []tests.Vault{rootVault, confVault.Vault})

				for _, v := range checkedVaults {
					err := enableKV(v)
					Expect(err).ToNot(HaveOccurred())
					_, err = v.RunVaultCmd("kv", "put", "kv/test", "my_key=my_value")
					Expect(err).ToNot(HaveOccurred())
					out, err := v.RunVaultCmd("kv", "get", "kv/test")
					Expect(err).ToNot(HaveOccurred())
					found := false
					for _, l := range strings.Split(string(out), "\n") {
						if strings.Contains(l, "my_key") && strings.Contains(l, "my_value") {
							found = true
						}
					}
					if !found {
						Fail(fmt.Sprintf("ouput of `kv get kv/test` doesn't contains 'my_key' or 'my_value': %s", string(out)))
					}
				}
			})
		})
	})
})

// enableKV enables kv version 1 at kv
func enableKV(v Vault) error {
	out, err := v.RunVaultCmd("secrets", "list")
	if err != nil {
		return err
	}
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "kv/") && strings.Contains(l, " kv ") {
			return nil
		}
	}
	_, err = v.RunVaultCmd("secrets", "enable", "-version=1", "kv")
	return err
}

func parse(vaultsB64Json string, originVaults []Vault) []Vault {
	data, err := b64.StdEncoding.DecodeString(vaultsB64Json)
	Expect(err).ToNot(HaveOccurred())
	var vaults []struct {
		vault.VaultConfiguration
		VaultToken string `structs:"token" json:"token"`
	}
	err = json.Unmarshal(data, &vaults)
	Expect(err).ToNot(HaveOccurred())
	var result []Vault
	for _, v := range vaults {
		originVault, err := findVaultByName(originVaults, v.VaultName)
		Expect(err).ToNot(HaveOccurred())
		result = append(result, Vault{
			ContainerID: originVault.ContainerID,
			Port:        originVault.Port,
			Name:        v.VaultName,
			Addr:        v.VaultUrl,
			Token:       v.VaultToken,
		})
	}
	return result
}

func findVaultByName(vaults []tests.Vault, name string) (*tests.Vault, error) {
	for _, v := range vaults {
		if v.Name == name {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("vault with name %q is not found among %v", name, vaults)
}

type Vault = tests.Vault

type ConfVault struct {
	Vault
	// ca of vault pki
	caPEM string
	// secretID for access into vault by flant_gitops
	secretID string
	// roleID for access into vault by flant_gitops
	roleID string
}

// StartAndConfigureConfVault runs conf vault
// activate and configure PKI at vault-cert-auth/roles/cert-auth
// activate approle and create secretID and roleID
func StartAndConfigureConfVault() (v ConfVault, err error) {
	vault, err := tests.RunAndWaitVaultUp("examples/conf", "vault-conf.hcl", "conf")

	Expect(err).ToNot(HaveOccurred())

	confVault := ConfVault{Vault: *vault}

	confVault.caPEM, err = applyPKI(confVault.Vault)
	if err != nil {
		return ConfVault{}, err
	}

	confVault.secretID, confVault.roleID, err = tests.GotSecretIDAndRoleIDatApprole(confVault.Vault)
	if err != nil {
		return ConfVault{}, err
	}

	return confVault, nil
}

// applyPKI enable PKI at conf vault - prepare everything for work flant_gitops and returns ca
func applyPKI(vault Vault) (caPEM string, err error) {
	_, err = vault.RunVaultCmd("secrets", "enable", "-path", "vault-cert-auth", "pki") // vault secrets enable -path=vault-cert-auth pki
	if err != nil {
		return
	}

	// vault write -field=certificate vault-cert-auth/root/generate/internal  \
	// common_name="negentropy" \
	// issuer_name="negentropy-2022"  \
	// ttl=87600h > negentropy_2022_ca.crt
	d, err := vault.RunVaultCmd("write", "-field=certificate", "vault-cert-auth/root/generate/internal", "common_name=negentropy", "issuer_name=negentropy-2022", "ttl=87600h")
	if err != nil {
		return
	}
	caPEM = string(d)

	_, err = vault.RunVaultCmd("write", "vault-cert-auth/roles/cert-auth", "allow_any_name=true",
		"max_ttl=1h") // vault write  vault-cert-auth/roles/cert-auth allow_any_name='true' max_ttl='1h'
	if err != nil {
		return
	}

	return caPEM, nil
}

// StartAndConfigureRootVault runs rootVault
// enabale auht/cert
// configure it by ca of conf_vault pki
func StartAndConfigureRootVault(confVaultPkiCa string) (v Vault, err error) {
	rootVault, err := tests.RunAndWaitVaultUp("examples/conf", "vault-root.hcl", "root")
	Expect(err).ToNot(HaveOccurred())

	_, err = rootVault.RunVaultCmd("policy", "write", "good", "etc/vault/good.hcl")
	if err != nil {
		return
	}

	_, err = rootVault.RunVaultCmd("auth", "enable", "cert") // vault auth enable cert
	if err != nil {
		return
	}

	// vault write auth/cert/certs/negentropy display_name='negentropy' policies='good' certificate='CA'
	_, err = rootVault.RunVaultCmd("write", "auth/cert/certs/negentropy",
		"display_name=negentropy", "policies=good", "certificate="+confVaultPkiCa)
	if err != nil {
		return
	}

	return *rootVault, nil
}

var backendLogLen int

func printNewLogs(logger *util.TestLogger) {
	logs := logger.GetLines()
	println("\n===LOGS:===")
	for _, l := range logs[backendLogLen:] {
		println(l)
	}
	backendLogLen = len(logs)
}
