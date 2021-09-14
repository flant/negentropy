package flow

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/auth_source"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/multipass"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	flant_vault_api "github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vault entities")
}

const (
	methodReaderOnlyPolicyName = "method_reader"
	methodReaderOnlyPolicy     = `
path "auth/flant_iam_auth/auth_method/*" {
  capabilities = ["read"]
}

path "auth/flant_iam_auth/issue/multipass_jwt/*" {
  capabilities = ["update"]
}

path "auth/token/renew" {
  capabilities = ["update"]
}
`
)

var (
	iamAuthClientWithRoot *api.Client
	iamClientWithRoot     *api.Client
	identityApi           *flant_vault_api.IdentityAPI
	sources               []auth_source.SourceForTest
	mountAccessorId       string
	jwtMethodName         string
	multipassMethodName   string
	tokenTTl              = 20 * time.Second

	authVaultAddr string
)

func uuidFromResp(resp *api.Secret, entityKey, key string) string {
	return resp.Data[entityKey].(map[string]interface{})[key].(string)
}

func createUser() *iam.User {
	tenantRaw, err := iamClientWithRoot.Logical().Write(lib.IamPluginPath+"/tenant", fixtures.RandomTenantCreatePayload())
	Expect(err).ToNot(HaveOccurred())
	tenantUUID := uuidFromResp(tenantRaw, "tenant", "uuid")

	userUUIDResp, err := iamClientWithRoot.Logical().Write(lib.IamPluginPath+"/tenant/"+tenantUUID+"/user/", fixtures.RandomUserCreatePayload())
	Expect(err).ToNot(HaveOccurred())
	userUUID := uuidFromResp(userUUIDResp, "user", "uuid")

	userRaw, err := iamClientWithRoot.Logical().Read(lib.IamPluginPath + "/tenant/" + tenantUUID + "/user/" + userUUID)
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
	_, err := iamClientWithRoot.Logical().Delete(lib.IamPluginPath + "/tenant/" + user.TenantUUID + "/user/" + user.UUID + "/multipass/" + multipass.UUID)
	Expect(err).ToNot(HaveOccurred())

	time.Sleep(2 * time.Second)
}

func prolongUserMultipass(positiveCase bool, uuid string, client *api.Client) string {
	maRaw, err := client.Logical().Write(lib.IamAuthPluginPath+"/issue/multipass_jwt/"+uuid, nil)
	if positiveCase {
		Expect(err).ToNot(HaveOccurred())
		return maRaw.Data["token"].(string)
	}

	Expect(err).To(HaveOccurred())

	return ""
}

func getJwks() *jose.JSONWebKeySet {
	jwksRaw, err := iamAuthClientWithRoot.Logical().Read(lib.IamAuthPluginPath + "/jwks/")
	Expect(err).ToNot(HaveOccurred())

	jwksStr, err := json.Marshal(jwksRaw.Data)
	Expect(err).ToNot(HaveOccurred())

	keySet := jose.JSONWebKeySet{}
	err = json.Unmarshal(jwksStr, &keySet)
	Expect(err).ToNot(HaveOccurred())

	return &keySet
}

func createUserMultipass(user *iam.User) (*iam.Multipass, string) {
	maRaw, err := iamClientWithRoot.Logical().Write(lib.IamPluginPath+"/tenant/"+user.TenantUUID+"/user/"+user.UUID+"/multipass", tools.ToMap(multipass.GetPayload()))
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
	tenantRaw, err := iamClientWithRoot.Logical().Write(lib.IamPluginPath+"/tenant", fixtures.RandomTenantCreatePayload())
	Expect(err).ToNot(HaveOccurred())
	tenantUUID := uuidFromResp(tenantRaw, "tenant", "uuid")

	saUUIDResp, err := iamClientWithRoot.Logical().Write(lib.IamPluginPath+"/tenant/"+tenantUUID+"/service_account/", fixtures.RandomServiceAccountCreatePayload())
	Expect(err).ToNot(HaveOccurred())
	saUUID := uuidFromResp(saUUIDResp, "service_account", "uuid")

	saRaw, err := iamClientWithRoot.Logical().Read(lib.IamPluginPath + "/tenant/" + tenantUUID + "/service_account/" + saUUID)
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

	_, err := iamAuthClientWithRoot.Logical().Write(lib.IamAuthPluginPath+"/auth_method/"+methodName, payload)
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

	_, err := iamAuthClientWithRoot.Logical().Write(lib.IamAuthPluginPath+"/auth_method/"+methodName, payload)
	Expect(err).ToNot(HaveOccurred())
}

func login(positiveCase bool, params map[string]interface{}) *api.SecretAuth {
	cl := configure.GetClientWithToken("", authVaultAddr)
	cl.ClearToken()

	secret, err := cl.Logical().Write(lib.IamAuthPluginPath+"/login", params)

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

func switchJwt(enable bool) {
	method := "enable"
	if !enable {
		method = "disable"
	}
	var err error
	_, err = iamClientWithRoot.Logical().Write(lib.IamPluginPath+"/jwt/"+method, nil)
	Expect(err).ToNot(HaveOccurred())

	_, err = iamAuthClientWithRoot.Logical().Write(lib.IamAuthPluginPath+"/jwt/"+method, nil)
	Expect(err).ToNot(HaveOccurred())

	time.Sleep(3 * time.Second)
}

var _ = BeforeSuite(func() {
	authVaultToken := lib.GetAuthRootToken()
	authVaultAddr = lib.GetAuthVaultUrl()
	iamAuthClientWithRoot = configure.GetClientWithToken(authVaultToken, authVaultAddr)

	rootVaultToken := lib.GetRootRootToken()
	rootVaultAddr := lib.GetRootVaultUrl()
	iamClientWithRoot = configure.GetClientWithToken(rootVaultToken, rootVaultAddr)

	identityApi = flant_vault_api.NewIdentityAPI(iamAuthClientWithRoot, hclog.NewNullLogger())

	role := configure.CreateGoodRole(iamAuthClientWithRoot)

	configure.ConfigureVaultAccess(iamAuthClientWithRoot, lib.IamAuthPluginPath, role)

	var err error
	switchJwt(true)

	sources = auth_source.GenerateSources()

	for _, s := range sources {
		p, name := s.ToPayload()
		_, err := iamAuthClientWithRoot.Logical().Write(lib.IamAuthPluginPath+"/auth_source/"+name, p)
		Expect(err).ToNot(HaveOccurred())
	}

	configure.CreatePolicy(iamAuthClientWithRoot, methodReaderOnlyPolicyName, methodReaderOnlyPolicy)

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
		return iamAuthClientWithRoot, nil
	}, "flant_iam_auth/").MountAccessor()
	Expect(err).ToNot(HaveOccurred())
	Expect(mountAccessorId).ToNot(BeEmpty())
})
