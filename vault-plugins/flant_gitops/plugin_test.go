package flant_gitops

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault entities")
}

var _ = Describe("flant_gitops", func() {
	var confVault ConfVault
	var rootVault Vault

	BeforeSuite(func() {
		var err error
		confVault, err = StartAndConfigureConfVault()
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("%#v\n", confVault)

		rootVault, err = StartAndConfigureRootVault(confVault.caPEM)

		time.Sleep(time.Minute * 5)
		// Запустим и настроим два волта: conf и root
		// Оба волта на https
		// conf :
		// - pki +
		// - выпуск серта +
		// - approle +

		// root :
		// auht/cert
		// скормить туда ca от  conf

		// подготовить пустую репу c одним комитом
		// запустить flant_gitops c моком системных часов и моком кубера
	}, 1.0)

	Describe("flant_gitops configuring", func() {
		// во  flant_gitops:
		// - self_access на conf
		// - configure/git_repository - включая первый комит из репы
		// - configure/vaults
	})

	Describe("flant_gitoos run peridoic functions without any new commits", func() {
		// здесь подвинем часы, дождёмся по логам периодик ран, поймём что ни хрена
	})

	Describe("flant_gitoos run peridoic functions with new commit", func() {
		// Запилим новый комит в репу
		// здесь подвинем часы, дождёмся по логам периодик ран, поймем что джоба поехала
		// посмотрим что там джоба прочитала
		// снова дождёмся по логам периодик ран, поймем что было обращение с проверкой джобы
		fmt.Printf("\n%#v\n", rootVault)
	})

})

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

func RunVaultCommandAtVault(vault Vault, args ...string) ([]byte, error) {
	err := os.Setenv("VAULT_ADDR", vault.addr)
	if err != nil {
		return nil, err
	}
	err = os.Setenv("VAULT_TOKEN", vault.token)
	if err != nil {
		return nil, err
	}
	err = os.Setenv("VAULT_CACERT", "examples/conf/ca.crt")
	if err != nil {
		return nil, err
	}
	output, err := RunVaultCommandWithError(args...)
	if err != nil {
		return nil, err
	}
	return output, nil
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

	//vault write -field=certificate vault-cert-auth/root/generate/internal  \
	//common_name="negentropy" \
	//issuer_name="negentropy-2022"  \
	//ttl=87600h > negentropy_2022_ca.crt
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

// runVaultAndWaitVaultUp run vault with specified config at specified port
// port should be the same as at the config
func runAndWaitVaultUp(configPath string, port string, name string) Vault {
	token := "root"
	go func() {
		if _, err := RunVaultCommandWithError("server", "-dev", "-config", configPath,
			"-dev-root-token-id", token); err != nil {
			panic(fmt.Sprintf("vault dev server failed: %s", err))
		}
	}()
	vault := Vault{
		name:  name,
		addr:  "https://127.0.0.1:" + port,
		token: token,
	}

	for {
		time.Sleep(1 * time.Second)
		if _, err := RunVaultCommandAtVault(vault, "status"); err != nil {
			continue
		}
		break
	}
	return vault
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
	_, err = RunVaultCommandAtVault(vault, "write", "auth/approle/role/good", "secret_id_ttl=30m", "token_ttl=90s", "token_policies=good")
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
		"display_name=negentropy", "policies=good", "certificate", confVaultPkiCa)
	if err != nil {
		return
	}

	return rootVault, nil
}
