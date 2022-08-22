package access_token_or_sapass_auth

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"time"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
)

var rootVaultAddr = lib.GetRootVaultUrl()

const tenantReadRole = "tenant.read"

var _ = Describe("Process of renew vst", func() {
	var flantIamSuite flant_iam_preparing.Suite
	flantIamSuite.BeforeSuite()
	cfg := flantIamSuite.PrepareForLoginTesting()
	_, cfg.UserMultipassJWT = specs.CreateUserMultipass(lib.NewUserMultipassAPI(lib.NewConfiguredIamVaultClient()),
		cfg.User, "test", 100*time.Second, 1000*time.Second, []string{flant_iam_preparing.SshRole})
	err := flantIamSuite.WaitPrepareForLoginTesting(cfg, 40)
	Expect(err).ToNot(HaveOccurred())

	Context("multipass auth", func() {
		rbForMultipass := createRolebinding(cfg.User, flant_iam_preparing.SshRole, cfg.Project.UUID)
		var vst string
		It("getting VST", func() {
			vst = Login(true, map[string]interface{}{
				"method": "multipass", "jwt": cfg.UserMultipassJWT,
				"roles": []map[string]interface{}{
					{"role": flant_iam_preparing.SshRole, "tenant_uuid": cfg.Tenant.UUID, "project_uuid": cfg.Project.UUID},
				},
			}, rootVaultAddr, 50).ClientToken
		})
		It("can renew VST", func() {
			vst = RenewVST(true, vst, rootVaultAddr, 50).ClientToken
		})
		It("can't renew after changing rolebinding", func() {
			rbForMultipass = updateRolebindingDescription(rbForMultipass, "rolebinding for test renew vst - updated")
			RenewVST(false, vst, rootVaultAddr, 100)
		})

		It("can't renew after deleting rolebinding", func() {
			vst = Login(true, map[string]interface{}{
				"method": "multipass", "jwt": cfg.UserMultipassJWT,
				"roles": []map[string]interface{}{
					{"role": flant_iam_preparing.SshRole, "tenant_uuid": cfg.Tenant.UUID, "project_uuid": cfg.Project.UUID},
				},
			}, rootVaultAddr, 50).ClientToken
			vst = RenewVST(true, vst, rootVaultAddr, 50).ClientToken

			lib.NewRoleBindingAPI(lib.NewConfiguredIamVaultClient()).Delete(tools.Params{
				"tenant":       cfg.Tenant.UUID,
				"role_binding": rbForMultipass.UUID,
			}, nil)
			RenewVST(false, vst, rootVaultAddr, 50)
		})
	})

	Context("okta-jwt auth", func() {
		rbForOktaJWT := createRolebinding(cfg.User, tenantReadRole, cfg.Project.UUID)
		var vst string
		accessToken, err := tools.GetOIDCAccessToken(cfg.User.UUID, cfg.User.Email)
		Expect(err).ToNot(HaveOccurred())

		It("getting VST", func() {
			vst = Login(true, map[string]interface{}{
				"method": "okta-jwt", "jwt": accessToken,
				"roles": []map[string]interface{}{
					{"role": tenantReadRole, "tenant_uuid": cfg.Tenant.UUID},
				},
			}, rootVaultAddr, 50).ClientToken
		})
		It("can renew VST", func() {
			vst = RenewVST(true, vst, rootVaultAddr, 50).ClientToken
		})
		It("can't renew after changing rolebinding", func() {
			rbForOktaJWT = updateRolebindingDescription(rbForOktaJWT, "rolebinding for test renew vst - updated")
			RenewVST(false, vst, rootVaultAddr, 100)
		})

		It("can't renew after deleting rolebinding", func() {
			accessToken, err := tools.GetOIDCAccessToken(cfg.User.UUID, cfg.User.Email)
			Expect(err).ToNot(HaveOccurred())
			vst = Login(true, map[string]interface{}{
				"method": "okta-jwt", "jwt": accessToken,
				"roles": []map[string]interface{}{
					{"role": tenantReadRole, "tenant_uuid": cfg.Tenant.UUID},
				},
			}, rootVaultAddr, 50).ClientToken
			vst = RenewVST(true, vst, rootVaultAddr, 50).ClientToken

			lib.NewRoleBindingAPI(lib.NewConfiguredIamVaultClient()).Delete(tools.Params{
				"tenant":       cfg.Tenant.UUID,
				"role_binding": rbForOktaJWT.UUID,
			}, nil)
			RenewVST(false, vst, rootVaultAddr, 50)
		})
	})
})

func createRolebinding(user model.User, role string, projectUUID string) model.RoleBinding {
	rolebinding := specs.CreateRoleBinding(lib.NewRoleBindingAPI(lib.NewConfiguredIamVaultClient()),
		model.RoleBinding{
			TenantUUID:  user.TenantUUID,
			Version:     "",
			Description: "rolebinding for test renew vst",
			ValidTill:   10_000_000_000,
			RequireMFA:  false,
			Members:     []model.MemberNotation{{Type: "user", UUID: user.UUID}},
			Projects:    []string{projectUUID},
			AnyProject:  false,
			Roles:       []model.BoundRole{{Name: role, Options: map[string]interface{}{"max_ttl": "1600m", "ttl": "800m"}}},
		})
	return rolebinding
}

func updateRolebindingDescription(rb model.RoleBinding, newDesc string) model.RoleBinding {
	params := tools.Params{
		"tenant":       rb.TenantUUID,
		"role_binding": rb.UUID,
	}
	bytes, _ := json.Marshal(rb)
	var updatePayload map[string]interface{}
	json.Unmarshal(bytes, &updatePayload) //nolint:errcheck
	delete(updatePayload, "valid_till")
	updatePayload["ttl"] = rb.ValidTill - time.Now().Unix()
	updatePayload["description"] = newDesc
	updateData := lib.NewRoleBindingAPI(lib.NewConfiguredIamVaultClient()).Update(params, url.Values{}, updatePayload)
	rawRoleBinding := updateData.Get("role_binding")
	data := []byte(rawRoleBinding.String())
	var rbd usecase.DenormalizedRoleBinding
	err := json.Unmarshal(data, &rbd)
	Expect(err).ToNot(HaveOccurred())

	return model.RoleBinding{
		ArchiveMark:     rbd.ArchiveMark,
		UUID:            rbd.UUID,
		TenantUUID:      rbd.TenantUUID,
		Version:         rbd.Version,
		Description:     rbd.Description,
		ValidTill:       rbd.ValidTill,
		RequireMFA:      rbd.RequireMFA,
		Users:           nil,
		Groups:          nil,
		ServiceAccounts: nil,
		Members:         specs.MapMembers(rbd.Members),
		AnyProject:      rbd.AnyProject,
		Projects:        specs.MapProjects(rbd.Projects),
		Roles:           rbd.Roles,
		Origin:          rbd.Origin,
		Extensions:      nil,
	}
}

func Login(positiveCase bool, params map[string]interface{}, vaultAddr string, attempts int) *api.SecretAuth {
	cl := configure.GetClientWithToken("", vaultAddr)
	cl.ClearToken()
	return checkSecretResult(func() (*api.Secret, error) {
		return cl.Logical().Write("auth/flant/login", params)
	}, positiveCase, attempts)
}

func RenewVST(positiveCase bool, vst string, vaultAddr string, attempts int) *api.SecretAuth {
	cl := configure.GetClientWithToken(vst, vaultAddr)
	return checkSecretResult(func() (*api.Secret, error) {
		return cl.Logical().Write("auth/token/renew-self", nil)
	}, positiveCase, attempts)
}

func checkSecretResult(f func() (*api.Secret, error), positiveCase bool, attempts int) *api.SecretAuth {
	var err error
	var secret *api.Secret
	for i := 0; i < attempts; i++ {
		secret, err = f()
		if positiveCase &&
			err == nil && secret != nil && secret.Auth != nil {
			break
		} else if !positiveCase && err != nil {
			break
		}
		time.Sleep(time.Millisecond * 20)
	}
	if positiveCase {
		Expect(err).ToNot(HaveOccurred())
		Expect(secret).ToNot(BeNil())
		Expect(secret.Auth).ToNot(BeNil())
		return secret.Auth
	} else {
		Expect(err).To(HaveOccurred())
	}
	return nil
}
