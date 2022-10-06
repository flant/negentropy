package flant_gitops

import (
	"bufio"
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/vault/command"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/sdk/physical"
	physInmem "github.com/hashicorp/vault/sdk/physical/inmem"
	"github.com/mitchellh/cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/vault"
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
	}, 1.0)

	Describe("flant_gitops configuring and running ", func() {
		It("configure self_access into conf vault", func() {
			req := &logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "configure_vault_access",
				Data: map[string]interface{}{
					"vault_addr":            confVault.addr,
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
				"name":         rootVault.name,
				"url":          rootVault.addr,
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
				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectWorkingCommits(ctx, b.Storage)
				Expect(err).ToNot(HaveOccurred())
				Expect(lastStartedCommit).To(Equal(""))
				Expect(lastPushedToK8sCommit).To(Equal(""))
				Expect(LastK8sFinishedCommit).To(Equal(""))

				err = b.B.PeriodicFunc(ctx, &logical.Request{Storage: b.Storage})
				Expect(err).ToNot(HaveOccurred())

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err = collectWorkingCommits(ctx, b.Storage)
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
				err := testGitRepo.WriteFileIntoRepoAndCommit("data", []byte("OUTPUT2\n"), "two")
				Expect(err).ToNot(HaveOccurred())

				err = b.B.PeriodicFunc(ctx, &logical.Request{Storage: b.Storage})
				Expect(err).ToNot(HaveOccurred())

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectWorkingCommits(ctx, b.Storage)
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

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectWorkingCommits(ctx, b.Storage)
				Expect(err).ToNot(HaveOccurred())
				Expect(lastStartedCommit).To(Equal(testGitRepo.CommitHashes[1])) // change is here
				Expect(lastPushedToK8sCommit).To(Equal(""))
				Expect(LastK8sFinishedCommit).To(Equal(""))
				Expect(b.MockKubeService.LenFinishedJobs()).To(Equal(0))

				for i := 0; i < 10; i++ { // waiting finishing task
					if b.MockKubeService.LenActiveJobs() > 0 {
						break
					}
					time.Sleep(time.Millisecond * 100)
				}
				Expect(b.MockKubeService.LenActiveJobs()).To(Equal(1))                           // change is here
				Expect(b.MockKubeService.HasActiveJob(testGitRepo.CommitHashes[1])).To(BeTrue()) // change is here
				printNewLogs(b.Logger)
			})

			It("flant_gitops run periodic functions after pushing  task to k8s", func() {
				ctx := context.Background()

				err := b.B.PeriodicFunc(ctx, &logical.Request{Storage: b.Storage})
				Expect(err).ToNot(HaveOccurred())

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectWorkingCommits(ctx, b.Storage)
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

				lastStartedCommit, lastPushedToK8sCommit, LastK8sFinishedCommit, err := collectWorkingCommits(ctx, b.Storage)
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

				checkedVaults := parse(job.VaultsB64Json)

				for _, v := range checkedVaults {
					err := enableKV(v)
					Expect(err).ToNot(HaveOccurred())
					_, err = RunVaultCommandAtVault(v, "kv", "put", "kv/test", "my_key=my_value")
					Expect(err).ToNot(HaveOccurred())
					out, err := RunVaultCommandAtVault(v, "kv", "get", "kv/test")
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
	out, err := RunVaultCommandAtVault(v, "secrets", "list")
	if err != nil {
		return err
	}
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "kv/") && strings.Contains(l, " kv ") {
			return nil
		}
	}
	_, err = RunVaultCommandAtVault(v, "secrets", "enable", "-version=1", "kv")
	return err
}

func parse(vaultsB64Json string) []Vault {
	data, err := b64.StdEncoding.DecodeString(vaultsB64Json)
	Expect(err).ToNot(HaveOccurred())
	vaultsStr := strings.Trim(string(data), "'")
	var vaults []struct {
		vault.VaultConfiguration
		VaultToken string `structs:"token" json:"token"`
	}
	err = json.Unmarshal([]byte(vaultsStr), &vaults)
	Expect(err).ToNot(HaveOccurred())
	var result []Vault
	for _, v := range vaults {
		result = append(result, Vault{
			name:  v.VaultName,
			addr:  v.VaultUrl,
			token: v.VaultToken,
		})
	}
	return result
}

type Vault struct {
	name  string
	addr  string
	token string
}

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
	vault := runAndWaitVaultUp("examples/conf/vault-conf.hcl", "8201", "conf")

	confVault := ConfVault{Vault: vault}

	confVault.caPEM, err = applyPKI(confVault.Vault)
	if err != nil {
		return ConfVault{}, err
	}

	confVault.secretID, confVault.roleID, err = gotSecretIDAndRoleIDatApprole(confVault.Vault)
	if err != nil {
		return ConfVault{}, err
	}

	return confVault, nil
}

// applyPKI enable PKI at conf vault - prepare everything for work flant_gitops and returns ca
func applyPKI(vault Vault) (caPEM string, err error) {
	_, err = RunVaultCommandAtVault(vault, "secrets", "enable", "-path", "vault-cert-auth", "pki") // vault secrets enable -path=vault-cert-auth pki
	if err != nil {
		return
	}

	// vault write -field=certificate vault-cert-auth/root/generate/internal  \
	// common_name="negentropy" \
	// issuer_name="negentropy-2022"  \
	// ttl=87600h > negentropy_2022_ca.crt
	d, err := RunVaultCommandAtVault(vault, "write", "-field=certificate", "vault-cert-auth/root/generate/internal", "common_name=negentropy", "issuer_name=negentropy-2022", "ttl=87600h")
	if err != nil {
		return
	}
	caPEM = string(d)

	_, err = RunVaultCommandAtVault(vault, "write", "vault-cert-auth/roles/cert-auth", "allow_any_name=true",
		"max_ttl=1h") // vault write  vault-cert-auth/roles/cert-auth allow_any_name='true' max_ttl='1h'
	if err != nil {
		return
	}

	return caPEM, nil
}

// unseal vault using output of init command, returns root token, collected from  initOut
func unseal(vault Vault, initOut []byte) (rootToken string) {
	outs := strings.Split(string(initOut), "\n")
	// remove garbage in case of debug
	for i := range outs {
		outs[i] = strings.ReplaceAll(outs[i], "\u001B[0m", "")
	}
	// collect keys
	if len(outs) < 5 {
		panic(fmt.Sprintf("not found 5 keys at:%s", string(initOut)))
	}
	shamir := []string{}
	for _, s := range outs[0:5] {
		aims := strings.Split(s, ":")
		if len(aims) == 2 {
			k := strings.TrimSpace(aims[1])
			shamir = append(shamir, k)
		}
	}
	if len(shamir) != 5 {
		panic(fmt.Sprintf("not found 5 keys at:%s", string(initOut)))
	}
	// unseal
	for _, k := range shamir {
		RunVaultCommandAtVault(vault, "operator", "unseal", k) //nolint:errcheck
	}
	// got root_key
	for _, s := range outs {
		if strings.Contains(s, "Initial Root Token") {
			aims := strings.Split(s, ":")
			rootToken = strings.TrimSpace(aims[1])
			return
		}
	}
	panic(fmt.Sprintf("not found Initial Root Token at:%s", string(initOut)))
}

// gotSecretIDAndRoleIDatApprole activates approle and returns secretID an roleID
func gotSecretIDAndRoleIDatApprole(vault Vault) (secretID string, roleID string, err error) {
	_, err = RunVaultCommandAtVault(vault, "auth", "enable", "approle")
	if err != nil {
		return
	}
	_, err = RunVaultCommandAtVault(vault, "policy", "write", "good", "examples/conf/good.hcl")
	if err != nil {
		return
	}
	_, err = RunVaultCommandAtVault(vault, "write", "auth/approle/role/good", "secret_id_ttl=360h", "token_ttl=15m", "token_policies=good")
	if err != nil {
		return
	}

	var responseData []byte
	var data map[string]interface{}
	{ // secretID
		responseData, err = RunVaultCommandAtVault(vault, "write", "-format", "json", "-f", "auth/approle/role/good/secret-id")
		if err != nil {
			return
		}

		if err = json.Unmarshal(responseData, &data); err != nil {
			return
		}

		secretID = data["data"].(map[string]interface{})["secret_id"].(string)

		fmt.Printf("Got secretID: %s\n", secretID)
	}

	{ // roleID
		responseData, err = RunVaultCommandAtVault(vault, "read", "-format", "json", "auth/approle/role/good/role-id")
		if err != nil {
			return
		}
		if err = json.Unmarshal(responseData, &data); err != nil {
			return
		}

		roleID = data["data"].(map[string]interface{})["role_id"].(string)

		fmt.Printf("Got roleID: %s\n", roleID)
	}
	return
}

// StartAndConfigureRootVault runs rootVault
// enabale auht/cert
// configure it by ca of conf_vault pki
func StartAndConfigureRootVault(confVaultPkiCa string) (v Vault, err error) {
	rootVault := runAndWaitVaultUp("examples/conf/vault-root.hcl", "8203", "root")

	_, err = RunVaultCommandAtVault(rootVault, "policy", "write", "good", "examples/conf/good.hcl")
	if err != nil {
		return
	}

	_, err = RunVaultCommandAtVault(rootVault, "auth", "enable", "cert") // vault auth enable cert
	if err != nil {
		return
	}

	// vault write auth/cert/certs/negentropy display_name='negentropy' policies='good' certificate='CA'
	_, err = RunVaultCommandAtVault(rootVault, "write", "auth/cert/certs/negentropy",
		"display_name=negentropy", "policies=good", "certificate="+confVaultPkiCa)
	if err != nil {
		return
	}

	return rootVault, nil
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

func RunVaultCommandAtVault(vault Vault, args ...string) ([]byte, error) {
	err := os.Setenv("VAULT_ADDR", vault.addr)
	// needs drop to valid work other staff from hashicorp
	defer os.Setenv("VAULT_ADDR", "") //nolint:errcheck
	if err != nil {
		return nil, err
	}
	err = os.Setenv("VAULT_TOKEN", vault.token)
	defer os.Setenv("VAULT_TOKEN", "") //nolint:errcheck
	if err != nil {
		return nil, err
	}
	err = os.Setenv("VAULT_CACERT", "examples/conf/ca.crt")
	defer os.Setenv("VAULT_CACERT", "") //nolint:errcheck
	if err != nil {
		return nil, err
	}
	output, err := RunVaultCommandWithError(args...)
	if err != nil {
		return nil, err
	}
	return output, nil
}

var vaultCLiMutex = &sync.Mutex{}

func RunVaultCommandWithError(args ...string) ([]byte, error) {
	vaultCLiMutex.Lock()
	defer vaultCLiMutex.Unlock()

	var output bytes.Buffer
	var errOutput bytes.Buffer

	opts := &command.RunOptions{
		Stdout: io.MultiWriter(os.Stdout, &output),
		Stderr: io.MultiWriter(os.Stderr, &errOutput),
	}

	rc := command.RunCustom(args, opts)
	if rc != 0 {
		return output.Bytes(), fmt.Errorf("vault failed with rc=%d:\n%s\n", rc, errOutput.String())
	}

	return output.Bytes(), nil
}

// runVaultAndWaitVaultUp run vault with specified config at specified port
// port should be the same as at the config
func runAndWaitVaultUp(configPath string, port string, name string) Vault {
	// this function is too complex due to data race problems
	vault := Vault{
		name: name,
		addr: "https://127.0.0.1:" + port,
	}
	tokenChan := make(chan string)
	go func() {
		// run init and unseal
		go func() {
			for {
				time.Sleep(1 * time.Second)
				d, err := RunVaultCommandWithError("operator", "init", "-address="+vault.addr, "-ca-cert=examples/conf/ca.crt")
				if err != nil {
					continue
				}
				tokenChan <- unseal(vault, d)
				break
			}
		}()
		srvcmd, output, errOutput := srvCmd()
		outcode := srvcmd.Run([]string{"-config", configPath})
		if outcode != 0 {
			panic(fmt.Sprintf("vault server failed: %d \noutput:\n%s \nerrOutput:\n %s", outcode, output.String(), errOutput.String()))
		}
	}()
	vault.token = <-tokenChan
	for {
		time.Sleep(1 * time.Second)
		if _, err := RunVaultCommandAtVault(vault, "status"); err != nil {
			continue
		}
		break
	}
	return vault
}

// srvCmd returns configured vault server command for running server and
// errOutput & output
func srvCmd() (*command.ServerCommand, *bytes.Buffer, *bytes.Buffer) {
	var output bytes.Buffer
	var errOutput bytes.Buffer

	runOpts := &command.RunOptions{
		Stdout: io.MultiWriter(&output),
		Stderr: io.MultiWriter(&errOutput),
	}

	serverCmdUi := &command.VaultUI{
		Ui: &cli.ColoredUi{
			ErrorColor: cli.UiColorRed,
			WarnColor:  cli.UiColorYellow,
			Ui: &cli.BasicUi{
				Reader: bufio.NewReader(os.Stdin),
				Writer: runOpts.Stdout,
			},
		},
	}
	srvcmd := &command.ServerCommand{
		BaseCommand: &command.BaseCommand{
			UI: serverCmdUi,
		},
		PhysicalBackends: map[string]physical.Factory{
			"inmem": physInmem.NewInmem,
		},
	}

	return srvcmd, &output, &errOutput
}
