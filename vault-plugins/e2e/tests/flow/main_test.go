package flow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/auth_source"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/configure"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/multipass"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/service_account"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tenant"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib/user"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	flant_vault_api "github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault entities")
}

const methodReaderOnlyPolicyName = "method_reader"
const methodReaderOnlyPolicy = `
path "auth/flant_iam_auth/auth_method/*" {
  capabilities = ["read"]
}
`

var (
	iamAuthClient       *api.Client
	iamClient           *api.Client
	identityApi         *flant_vault_api.IdentityAPI
	sources             []auth_source.SourceForTest
	mountAccessorId     string
	jwtMethodName       string
	multipassMethodName string
	tokenTTl            = 5 * time.Second
)

func uuidFromResp(resp *api.Secret, entityKey, key string) string {
	return resp.Data[entityKey].(map[string]interface{})[key].(string)
}

func createUser() *iam.User {
	tenantRaw, err := iamClient.Logical().Write(lib.IamPluginPath+"/tenant", tools.ToMap(tenant.GetPayload()))
	Expect(err).ToNot(HaveOccurred())
	tenantUUID := uuidFromResp(tenantRaw, "tenant", "uuid")

	userUUIDResp, err := iamClient.Logical().Write(lib.IamPluginPath+"/tenant/"+tenantUUID+"/user/", tools.ToMap(user.GetPayload()))
	Expect(err).ToNot(HaveOccurred())
	userUUID := uuidFromResp(userUUIDResp, "user", "uuid")

	userRaw, err := iamClient.Logical().Read(lib.IamPluginPath + "/tenant/" + tenantUUID + "/user/" + userUUID)
	Expect(err).ToNot(HaveOccurred())

	userObj := iam.User{}
	js, err := json.Marshal(userRaw.Data["user"])
	Expect(err).ToNot(HaveOccurred())
	err = json.Unmarshal(js, &userObj)
	Expect(err).ToNot(HaveOccurred())

	// need wait for sync with iam_auth
	// todo need waitFor function?
	time.Sleep(5 * time.Second)

	return &userObj
}

func deleteUserMultipass(user *iam.User, multipass *iam.Multipass) {
	_, err := iamClient.Logical().Delete(lib.IamPluginPath+"/tenant/"+user.TenantUUID+"/user/"+user.UUID+"/multipass/"+multipass.UUID)
	Expect(err).ToNot(HaveOccurred())

	time.Sleep(2 * time.Second)
}

func createUserMultipass(user *iam.User) (*iam.Multipass, string) {
	maRaw, err := iamClient.Logical().Write(lib.IamPluginPath+"/tenant/"+user.TenantUUID+"/user/"+user.UUID+"/multipass", tools.ToMap(multipass.GetPayload()))
	Expect(err).ToNot(HaveOccurred())

	maObj := iam.Multipass{}
	js, err := json.Marshal(maRaw.Data["multipass"])
	Expect(err).ToNot(HaveOccurred())
	err = json.Unmarshal(js, &maObj)
	Expect(err).ToNot(HaveOccurred())

	// todo verify is jwt
	token := maRaw.Data["token"].(string)
	Expect(token).ToNot(BeEmpty())

	// need wait for sync with iam_auth
	// todo need waitFor function?
	time.Sleep(5 * time.Second)

	return &maObj, token
}

func createServiceAccount() *iam.ServiceAccount {
	tenantRaw, err := iamClient.Logical().Write(lib.IamPluginPath+"/tenant", tools.ToMap(tenant.GetPayload()))
	Expect(err).ToNot(HaveOccurred())
	tenantUUID := uuidFromResp(tenantRaw, "tenant", "uuid")

	saUUIDResp, err := iamClient.Logical().Write(lib.IamPluginPath+"/tenant/"+tenantUUID+"/service_account/", tools.ToMap(service_account.GetPayload()))
	Expect(err).ToNot(HaveOccurred())
	saUUID := uuidFromResp(saUUIDResp, "service_account", "uuid")

	saRaw, err := iamClient.Logical().Read(lib.IamPluginPath + "/tenant/" + tenantUUID + "/service_account/" + saUUID)
	Expect(err).ToNot(HaveOccurred())

	saObj := iam.ServiceAccount{}
	js, err := json.Marshal(saRaw.Data["service_account"])
	Expect(err).ToNot(HaveOccurred())
	err = json.Unmarshal(js, &saObj)
	Expect(err).ToNot(HaveOccurred())

	// need wait for sync with iam_auth
	// todo need waitFor function?
	time.Sleep(5 * time.Second)

	return &saObj
}

func createJwtAuthMethod(methodName, userClaim string, source auth_source.SourceForTest, payloadRewrite map[string]interface{}) {
	payload := map[string]interface{}{
		"bound_audiences": auth_source.Audience,
		"token_policies":  []string{"good"},
		"token_type":      "default",
		"token_ttl":       "1m",
		"method_type":     model.MethodTypeJWT,
		"source":          source.Source.Name,
		"user_claim":      userClaim,
	}

	if len(payloadRewrite) > 0 {
		for k, v := range payloadRewrite {
			payload[k] = v
		}
	}

	_, err := iamAuthClient.Logical().Write(lib.IamAuthPluginPath+"/auth_method/"+methodName, payload)
	Expect(err).ToNot(HaveOccurred())
}

func createMultipassAuthMethod(methodName string, payloadRewrite map[string]interface{}) {
	payload := map[string]interface{}{
		"token_policies": []string{"good"},
		"token_type":     "default",
		"token_ttl":      "1m",
		"method_type":    model.MethodTypeMultipass,
	}

	if len(payloadRewrite) > 0 {
		for k, v := range payloadRewrite {
			payload[k] = v
		}
	}

	_, err := iamAuthClient.Logical().Write(lib.IamAuthPluginPath+"/auth_method/"+methodName, payload)
	Expect(err).ToNot(HaveOccurred())
}

func login(positiveCase bool, params map[string]interface{}) *api.SecretAuth {
	secret, err := iamAuthClient.Logical().Write(lib.IamAuthPluginPath+"/login", params)
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

var _ = BeforeSuite(func() {
	token := lib.GetSecondRootToken()
	iamAuthClient = configure.GetClient(token)
	iamAuthClient.SetToken(token)

	iamClient = configure.GetClient(token)
	iamClient.SetToken(token)

	identityApi = flant_vault_api.NewIdentityAPI(configure.GetClient(token), hclog.NewNullLogger())

	role := configure.CreateGoodRole(token)

	configure.ConfigureVaultAccess(token, lib.IamAuthPluginPath, role)

	var err error
	_, err = iamClient.Logical().Write(lib.IamPluginPath+"/jwt/enable", nil)
	Expect(err).ToNot(HaveOccurred())

	_, err = iamAuthClient.Logical().Write(lib.IamAuthPluginPath+"/jwt/enable", nil)
	Expect(err).ToNot(HaveOccurred())

	sources = auth_source.GenerateSources()

	for _, s := range sources {
		p, name := s.ToPayload()
		_, err := iamAuthClient.Logical().Write(lib.IamAuthPluginPath+"/auth_source/"+name, p)
		Expect(err).ToNot(HaveOccurred())
	}

	configure.CreatePolicy(token, methodReaderOnlyPolicyName, methodReaderOnlyPolicy)

	jwtMethodName = tools.RandomStr()
	createJwtAuthMethod(jwtMethodName, "uuid", auth_source.JWTWithEaNameEmail, map[string]interface{}{
		"token_ttl":               tokenTTl.String(),
		"token_policies":          []string{methodReaderOnlyPolicyName},
		"token_no_default_policy": true,
	})

	multipassMethodName = tools.RandomStr()
	createMultipassAuthMethod(multipassMethodName, map[string]interface{}{
		"token_ttl":               tokenTTl.String(),
		"token_policies":          []string{methodReaderOnlyPolicyName},
		"token_no_default_policy": true,
	})

	mountAccessorId, err = vault.NewMountAccessorGetter(func() (*api.Client, error) {
		return configure.GetClient(token), nil
	}, "flant_iam_auth/").MountAccessor()
	Expect(err).ToNot(HaveOccurred())
	Expect(mountAccessorId).ToNot(BeEmpty())
})
