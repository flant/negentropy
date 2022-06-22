package access_token_or_sapass_auth

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
)

var rootVaultAddr = lib.GetRootVaultUrl()

var _ = Describe("Process of getting access through:", func() {
	var flantIamSuite flant_iam_preparing.Suite
	flantIamSuite.BeforeSuite()
	cfg := flantIamSuite.PrepareForLoginTesting()
	err := flantIamSuite.WaitPrepareForLoginTesting(cfg, 40)
	Expect(err).ToNot(HaveOccurred())

	Context("access_token auth", func() {

		It("fail with invalid token", func() {
			fakeToken := "eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiJodHRwOi8vb2lkYy1tb2NrOjk5OTgiLCJzdWIiOiJzdWJqZWN0XzIwMjEtMTItMTQgMTI6MjM6NDQuMTY0MjgyODQzICswMDAwIFVUQyBtPSsyODYzLjAxMTY2MTY5MyIsImF1ZCI6WyJhdWQ2NjYiXSwianRpIjoiaWQiLCJleHAiOjE2Mzk0ODQ5MjQsImlhdCI6MTYzOTQ4NDYyNCwibmJmIjoxNjM5NDg0NjI0LCJwcml2YXRlX2NsYWltIjoidGVzdCJ9.Mg9U-UciCjqqEeuu6SOKTfs36SpqciHM2ailkyWsVc0oSKxDQObivMPTtV04rD0PIqNe7Dp-2dmr9xqvfX8nFv-_TWthM-lhsknquPW-okM616KZf9lzjI08ZhzT1zksJYAu7Pz0dqSYYvirnu4MU3dPxmG16kzwwmhF13G01Is8s820wEkVgwWzi3FWJvu18cliovGd_5rwd4_hdDwKT3a_mfNEw8e7ZVC-l3irzmOstD56vsnwfOfprtKDUbnlOY9dDBd82gQ0jU7i8iLsQyYAJUrQb-uK0AX22fyIg-MFtj-TXUQF9PJ-3sOR4VrItu6Re65ZCZc0NvVvqbRv5w"
			tools.LoginAccessToken(false, map[string]interface{}{"method": "okta-jwt", "jwt": fakeToken}, rootVaultAddr)
		})
		It("fail with valid access_token issued by oidc, with invalid user email", func() {
			accessToken, err := tools.GetOIDCAccessToken(cfg.User.UUID, "fake_email@gmail.com")
			Expect(err).ToNot(HaveOccurred())
			tools.LoginAccessToken(false, map[string]interface{}{"method": "okta-jwt", "jwt": accessToken}, rootVaultAddr)
		})
		Context("getting VST against valid jwt of valid user", func() {
			accessToken, err := tools.GetOIDCAccessToken(cfg.User.UUID, cfg.User.Email)
			Expect(err).ToNot(HaveOccurred())

			vst := tools.LoginAccessToken(true, map[string]interface{}{"method": "okta-jwt", "jwt": accessToken}, rootVaultAddr).ClientToken
			It("getting access to tenant list at auth vault", func() {
				resp, err, statusCode := makeRequest(vst, "GET", rootVaultAddr, lib.IamAuthPluginPath+"/tenant/?list=true")
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(200))
				Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
			})
			It("permission denied to read tenant at auth vault", func() {
				_, err, statusCode := makeRequest(vst, "GET", rootVaultAddr, lib.IamPluginPath+"/tenant/"+cfg.Tenant.UUID)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(403))
			})
		})
		Context("getting VST with role against valid jwt of valid user and implementing web access", func() {
			accessToken, err := tools.GetOIDCAccessToken(cfg.User.UUID, cfg.User.Email)
			Expect(err).ToNot(HaveOccurred())
			vst := tools.LoginAccessToken(true, map[string]interface{}{
				"method": "okta-jwt", "jwt": accessToken,
				"roles": []map[string]interface{}{
					{"role": "tenants.list.auth"},
				},
			}, rootVaultAddr).ClientToken
			fmt.Printf("VST=%s\n", vst)

			It("getting access to tenant list at auth vault", func() {
				resp, err, statusCode := makeRequest(vst, "GET", rootVaultAddr, lib.IamAuthPluginPath+"/tenant/?list=true")
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(200))
				Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
			})
			It("getting access to vst_owner path", func() {
				resp, err, statusCode := makeRequest(vst, "GET", rootVaultAddr, lib.IamAuthPluginPath+"/vst_owner")
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(200))
				Expect(string(resp)).To(ContainSubstring("\"data\":{\"user\":{"))
				Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
				Expect(string(resp)).To(ContainSubstring(cfg.User.UUID))
				Expect(string(resp)).To(ContainSubstring(cfg.User.FullIdentifier))
				Expect(string(resp)).To(ContainSubstring(cfg.User.Identifier))
			})
			It("getting acccess to check_permissions", func() {
				resp, err := makeRequestWithPayload(vst, rootVaultAddr, lib.IamAuthPluginPath+"/check_permissions",
					map[string]interface{}{
						"method": "okta-jwt",
						"roles": []model.RoleClaim{
							{
								Role:       "tenant.read",
								TenantUUID: cfg.User.TenantUUID,
							},
							{
								Role: "tenants.list.auth",
							},
						}})

				Expect(err).ToNot(HaveOccurred())
				Expect(resp).To(HaveKey("permissions"))
				results := resp["permissions"]
				rawData, err := json.Marshal(results)
				Expect(err).ToNot(HaveOccurred())
				var permissions []authz.RoleClaimResult
				err = json.Unmarshal(rawData, &permissions)
				Expect(err).ToNot(HaveOccurred())
				Expect(permissions).To(HaveLen(2))
				permission1 := permissions[0]
				Expect(permission1.Role).To(Equal("tenant.read"))
				Expect(permission1.AllowLogin).To(BeFalse())
				Expect(permission1.RolebindingExists).To(BeFalse())
				permission2 := permissions[1]
				Expect(permission2.Role).To(Equal("tenants.list.auth"))
				Expect(permission2.AllowLogin).To(BeTrue())
				Expect(permission2.RolebindingExists).To(BeFalse())
			})
		})
	})

	Context("service_account password auth", func() {
		Context("succeed if valid service_account_password_uuid&secret", func() {
			var vst string
			It("getting VST", func() {
				vst = loginServiceAccountPass(true, map[string]interface{}{
					"method":                          "sapassword",
					"service_account_password_uuid":   cfg.ServiceAccountPassword.UUID,
					"service_account_password_secret": cfg.ServiceAccountPassword.Secret,
					"roles": []map[string]interface{}{
						{"role": "servers.register", "tenant_uuid": cfg.Tenant.UUID, "project_uuid": cfg.Project.UUID},
					},
				}).ClientToken
			})
			It("getting access to tenant list at auth vault, using vst", func() {
				resp, err, statusCode := makeRequest(vst, "GET", rootVaultAddr, lib.IamAuthPluginPath+"/tenant/?list=true")
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(200))
				Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
			})
		})
		Context("fail", func() {
			It("if invalid service_account_password_uuid", func() {
				loginServiceAccountPass(false, map[string]interface{}{
					"method":                          "sapassword",
					"service_account_password_uuid":   "deadhead-a886-44f3-82bd-334e5de75fe3",
					"service_account_password_secret": cfg.ServiceAccountPassword.Secret,
					"roles": []map[string]interface{}{
						{"role": "servers.register", "tenant_uuid": cfg.Tenant.UUID, "project_uuid": cfg.Project.UUID},
					},
				})
			})
			It("if invalid service_account_password_secret", func() {
				loginServiceAccountPass(false, map[string]interface{}{
					"method":                          "sapassword",
					"service_account_password_uuid":   cfg.ServiceAccountPassword.UUID,
					"service_account_password_secret": "InvalidSecret",
					"roles": []map[string]interface{}{
						{"role": "servers.register", "tenant_uuid": cfg.Tenant.UUID, "project_uuid": cfg.Project.UUID},
					},
				})
			})
		})
	})
})

func loginServiceAccountPass(positiveCase bool, params map[string]interface{}) *api.SecretAuth {
	cl := configure.GetClientWithToken("", rootVaultAddr)
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

func makeRequest(token string, method string, vaultUrl string, requestUrl string) ([]byte, error, int) {
	url := vaultUrl + "/v1/" + requestUrl

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err, 0
	}
	req.Header.Add("X-Vault-Token", token)

	res, err := client.Do(req)
	if err != nil {
		return nil, err, 0
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err, 0
	}
	return body, nil, res.StatusCode
}

func makeRequestWithPayload(token string, vaultUrl string, requestUrl string, payload map[string]interface{}) (map[string]interface{}, error) {
	cl := configure.GetClientWithToken(token, vaultUrl)
	secret, err := cl.Logical().Write(requestUrl, payload)

	if err != nil {
		return nil, err
	}

	return secret.Data, nil
}
