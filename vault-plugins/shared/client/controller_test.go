package client

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault entities")
}

var _ = Describe("VaultClientController", func() {
	// vault tls common certificate CA
	var vaultCaCert string

	// to this vault client should provide access
	var vault tests.Vault

	var err error
	vaultCaCrtData, err := ioutil.ReadFile("examples/conf/ca.crt") // it is common tls cert
	Expect(err).ToNot(HaveOccurred())
	vaultCaCert = string(vaultCaCrtData)

	// run and configure vault
	vault = StartAndConfigureVault()
	Expect(err).ToNot(HaveOccurred())

	Describe("Unconfigured start", func() {
		Describe("Empty storage state", func() {
			controller, err := NewAccessVaultClientController(&logical.InmemStorage{}, hclog.Default())
			Expect(err).ToNot(HaveOccurred())
			It("ApiClient() returns ErrNotSetConf", func() {
				_, err := controller.APIClient()
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrNotSetConf))
			})
			It("GetApiConfig(ctx) returns ErrNotSetConf", func() {
				_, err := controller.GetApiConfig(context.Background())
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrNotSetConf))
			})
			It("UpdateOutdated(ctx) doesn't return ErrNotSetConf, but has it in logs", func() {
				err := controller.UpdateOutdated(context.Background())
				Expect(err).ToNot(HaveOccurred())
				controllerInternal, ok := controller.(*VaultClientController)
				Expect(ok).To(BeTrue())
				Expect(controllerInternal.apiClient).To(BeNil(), "client should still be nil")
			})
		})

		Describe("state after successful run of HandleConfigureVaultAccess(...)", func() {
			controller, secretID, _ := configuredController(vault, vaultCaCert)

			It("GetApiConfig(ctx) returns valid config", func() {
				cfg, err := controller.GetApiConfig(context.Background())
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg).ToNot(BeNil())
				Expect(cfg.APIURL).To(Equal(vault.Addr))
			})

			It("vaultAccessConfig stored in storage has changed secret_id", func() {
				controllerInternal, ok := controller.(*VaultClientController)
				Expect(ok).To(BeTrue())
				cfg, err := controllerInternal.getVaultClientConfig(context.Background())
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg).ToNot(BeNil())
				Expect(cfg.APIURL).To(Equal(vault.Addr))
				Expect(cfg.SecretID).ToNot(Equal(secretID))
			})

			It("ApiClient() returns valid client which can access vault", func() {
				cl, err := controller.APIClient()
				Expect(err).ToNot(HaveOccurred())
				err = isClientValid(cl)
				Expect(err).ToNot(HaveOccurred())
			})

			It("after running UpdateOutdated(ctx) token of client should change", func() {
				cl, err := controller.APIClient()
				Expect(err).ToNot(HaveOccurred())
				oldToken := cl.Token()

				err = controller.UpdateOutdated(context.Background())
				Expect(err).ToNot(HaveOccurred())

				cl, err = controller.APIClient()
				Expect(err).ToNot(HaveOccurred())
				Expect(cl.Token()).ToNot(Equal(oldToken))
			})
		})

		Describe("state with valid config and invalid client", func() {
			controller, _, _ := configuredController(vault, vaultCaCert)
			revokeClientToken(controller) // revoke token - make client broken
			It("client starts failed requests", func() {
				cl, err := controller.APIClient()
				Expect(err).ToNot(HaveOccurred())
				err = isClientValid(cl)

				Expect(err).To(HaveOccurred())
			})
			It("after finishing UpdateOutdated(ctx), client became ok", func() {
				err = controller.UpdateOutdated(context.Background())
				Expect(err).ToNot(HaveOccurred())
				cl, err := controller.APIClient()
				Expect(err).ToNot(HaveOccurred())
				err = isClientValid(cl)

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Restart  after configuring", func() {
		var storage logical.Storage
		It("Run configured controller, for getting storage with configuration", func() {
			controller, _, _ := configuredController(vault, vaultCaCert)
			controllerInternal, ok := controller.(*VaultClientController)
			Expect(ok).To(BeTrue())
			cl, err := controller.APIClient()
			Expect(err).ToNot(HaveOccurred())
			err = isClientValid(cl)

			Expect(err).ToNot(HaveOccurred())

			storage = controllerInternal.storage
		})

		It("Create by constructor on storage with configuration inside - controller should be ready for using", func() {
			controller, err := NewAccessVaultClientController(storage, hclog.Default())
			Expect(err).ToNot(HaveOccurred())
			cl, err := controller.APIClient()
			Expect(err).ToNot(HaveOccurred())
			err = isClientValid(cl)

			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func revokeClientToken(controller AccessVaultClientController) {
	cl, err := controller.APIClient()
	Expect(err).ToNot(HaveOccurred())
	_, err = cl.Logical().Write("/auth/token/revoke-self", nil)
	Expect(err).ToNot(HaveOccurred())
}

func isClientValid(cl *api.Client) error {
	_, err := cl.Logical().Read("sys/mounts")
	if err != nil {
		return err
	}
	return nil
}

// StartAndConfigureVault runs vault
func StartAndConfigureVault() tests.Vault {
	return tests.RunAndWaitVaultUp("examples/conf/vault.hcl", "8203", "root")
}

type (
	secretID = string
	roleID   = string
)

func configuredController(vault tests.Vault, vaultsCaCert string) (AccessVaultClientController, secretID, roleID) {
	secretID, roleID, err := tests.GotSecretIDAndRoleIDatApprole(vault)
	Expect(err).ToNot(HaveOccurred())
	controller, err := NewAccessVaultClientController(&logical.InmemStorage{}, hclog.Default())
	Expect(err).ToNot(HaveOccurred())
	resp, err := controller.HandleConfigureVaultAccess(context.Background(), nil, &framework.FieldData{
		Raw: map[string]interface{}{
			"vault_addr":            vault.Addr,
			"vault_tls_server_name": "vault_host",
			"role_name":             "good",
			"secret_id_ttl":         "360h",
			"approle_mount_point":   "auth/approle/",
			"secret_id":             secretID,
			"role_id":               roleID,
			"vault_cacert":          vaultsCaCert,
		},
		Schema: PathConfigure(controller).Fields,
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(resp.Data).To(BeNil(), "path_configure should return empty data here")
	return controller, secretID, roleID
}
