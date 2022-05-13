package tools

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib/configure"
)

func LoginAccessToken(positiveCase bool, params map[string]interface{}, vaultAddr string) *api.SecretAuth {
	cl := configure.GetClientWithToken("", vaultAddr)
	cl.ClearToken()

	secret, err := cl.Logical().Write("auth/flant/login", params)

	if positiveCase {
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(secret).ToNot(gomega.BeNil())
		gomega.Expect(secret.Auth).ToNot(gomega.BeNil())

		return secret.Auth
	} else {
		gomega.Expect(err).To(gomega.HaveOccurred())
	}
	return nil
}

func GetOIDCAccessToken(userUUID string) (string, error) {
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
