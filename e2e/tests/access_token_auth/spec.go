package access_token_auth

import (
	_ "embed"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/configure"
	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
)

var authVaultAddr string = os.Getenv("AUTH_VAULT_URL")

var _ = Describe("Process of running access through access_token", func() {
	var flantIamSuite flant_iam_preparing.Suite
	flantIamSuite.BeforeSuite()
	cfg := flantIamSuite.PrepareForAccessTokenTesting()

	It("fail with invalid token", func() {
		fakeToken := "eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiJodHRwOi8vb2lkYy1tb2NrOjk5OTgiLCJzdWIiOiJzdWJqZWN0XzIwMjEtMTItMTQgMTI6MjM6NDQuMTY0MjgyODQzICswMDAwIFVUQyBtPSsyODYzLjAxMTY2MTY5MyIsImF1ZCI6WyJhdWQ2NjYiXSwianRpIjoiaWQiLCJleHAiOjE2Mzk0ODQ5MjQsImlhdCI6MTYzOTQ4NDYyNCwibmJmIjoxNjM5NDg0NjI0LCJwcml2YXRlX2NsYWltIjoidGVzdCJ9.Mg9U-UciCjqqEeuu6SOKTfs36SpqciHM2ailkyWsVc0oSKxDQObivMPTtV04rD0PIqNe7Dp-2dmr9xqvfX8nFv-_TWthM-lhsknquPW-okM616KZf9lzjI08ZhzT1zksJYAu7Pz0dqSYYvirnu4MU3dPxmG16kzwwmhF13G01Is8s820wEkVgwWzi3FWJvu18cliovGd_5rwd4_hdDwKT3a_mfNEw8e7ZVC-l3irzmOstD56vsnwfOfprtKDUbnlOY9dDBd82gQ0jU7i8iLsQyYAJUrQb-uK0AX22fyIg-MFtj-TXUQF9PJ-3sOR4VrItu6Re65ZCZc0NvVvqbRv5w"
		login(false, map[string]interface{}{"method": "oidc-mock-access-token", "jwt": fakeToken})
	})
	It("fail with valid access_token issued by oidc, with invalid user uuid", func() {
		accessToken, err := getAccessToken("00000001-0001-4001-A001-000000000001")
		Expect(err).ToNot(HaveOccurred())
		login(false, map[string]interface{}{"method": "oidc-mock-access-token", "jwt": accessToken})
	})
	Context("getting VST against valid jwt of vaild user", func() {
		accessToken, err := getAccessToken(cfg.User.UUID)
		Expect(err).ToNot(HaveOccurred())
		time.Sleep(time.Second * 5)
		vst := login(true, map[string]interface{}{"method": "oidc-mock-access-token", "jwt": accessToken}).ClientToken
		It("getting access to tenant list at auth vault", func() {
			resp, err, statusCode := makeRequest(vst, "GET", authVaultAddr, lib.IamAuthPluginPath+"/tenant/?list=true")
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(200))
			Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
		})
		It("permission denied to read tenant at auth vault", func() {
			_, err, statusCode := makeRequest(vst, "GET", authVaultAddr, lib.IamAuthPluginPath+"/tenant/"+cfg.Tenant.UUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(403))
		})
	})
	Context("getting VST with flant_iam against valid jwt of vaild user", func() {
		accessToken, err := getAccessToken(cfg.User.UUID)
		Expect(err).ToNot(HaveOccurred())
		vst := login(true, map[string]interface{}{
			"method": "oidc-mock-access-token", "jwt": accessToken,
			"roles": []map[string]interface{}{
				{"role": "flant_iam"},
			},
		}).ClientToken
		It("getting access to tenant list at auth vault", func() {
			resp, err, statusCode := makeRequest(vst, "GET", authVaultAddr, lib.IamAuthPluginPath+"/tenant/?list=true")
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(200))
			Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
		})
		It("getting access  to read tenant at auth vault", func() {
			resp, err, statusCode := makeRequest(vst, "GET", authVaultAddr, lib.IamAuthPluginPath+"/tenant/"+cfg.Tenant.UUID)
			Expect(err).ToNot(HaveOccurred())
			Expect(statusCode).To(Equal(200))
			Expect(string(resp)).To(ContainSubstring(cfg.Tenant.UUID))
		})
	})
})

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
