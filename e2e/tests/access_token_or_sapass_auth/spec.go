package access_token_or_sapass_auth

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
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
			loginAccessToken(false, map[string]interface{}{"method": "oidc-mock-access-token", "jwt": fakeToken})
		})
		It("fail with valid access_token issued by oidc, with invalid user uuid", func() {
			accessToken, err := getAccessToken("00000001-0001-4001-A001-000000000001")
			Expect(err).ToNot(HaveOccurred())
			loginAccessToken(false, map[string]interface{}{"method": "oidc-mock-access-token", "jwt": accessToken})
		})
		Context("getting VST against valid jwt of vaild user", func() {
			accessToken, err := getAccessToken(cfg.User.UUID)
			Expect(err).ToNot(HaveOccurred())

			vst := loginAccessToken(true, map[string]interface{}{"method": "oidc-mock-access-token", "jwt": accessToken}).ClientToken
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
		Context("getting VST with flant_iam against valid jwt of vaild user", func() {
			accessToken, err := getAccessToken(cfg.User.UUID)
			Expect(err).ToNot(HaveOccurred())
			vst := loginAccessToken(true, map[string]interface{}{
				"method": "oidc-mock-access-token", "jwt": accessToken,
				"roles": []map[string]interface{}{
					{"role": "iam_read", "tenant_uuid": cfg.Tenant.UUID},
				},
			}).ClientToken
			fmt.Printf("VST=%s", vst)
			It("getting access to tenant list at auth vault", func() {
				resp, err, statusCode := makeRequest(vst, "GET", rootVaultAddr, lib.IamAuthPluginPath+"/tenant/?list=true")
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(200))
				Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
			})
			It("getting access to read tenant at auth vault", func() {
				resp, err, statusCode := makeRequest(vst, "GET", rootVaultAddr, lib.IamPluginPath+"/tenant/"+cfg.Tenant.UUID)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusCode).To(Equal(200))
				Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
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
						{"role": "register_server", "tenant_uuid": cfg.Tenant.UUID, "project_uuid": cfg.Project.UUID},
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
						{"role": "register_server", "tenant_uuid": cfg.Tenant.UUID, "project_uuid": cfg.Project.UUID},
					},
				})
			})
			It("if invalid service_account_password_secret", func() {
				loginServiceAccountPass(false, map[string]interface{}{
					"method":                          "sapassword",
					"service_account_password_uuid":   cfg.ServiceAccountPassword.UUID,
					"service_account_password_secret": "InvalidSecret",
					"roles": []map[string]interface{}{
						{"role": "register_server", "tenant_uuid": cfg.Tenant.UUID, "project_uuid": cfg.Project.UUID},
					},
				})
			})
		})
	})
})

func loginAccessToken(positiveCase bool, params map[string]interface{}) *api.SecretAuth {
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

func getAccessToken(userUUID string) (string, error) {
	url := "http://localhost:9998/custom_access_token?uuid=" + userUUID
	method := "GET"

	client := &http.Client{}

	req, err := http.NewRequest(method, url, strings.NewReader(""))
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		if err != nil {
			return "", err
		}
	}
	return string(body), nil
}

func makeRequest(token string, method string, vault_url string, request_url string) ([]byte, error, int) {
	url := vault_url + "/v1/" + request_url

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
