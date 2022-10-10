package lib

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/flant/negentropy/vault-plugins/shared/tests"

	. "github.com/onsi/ginkgo"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
)

type customHeadersTransport struct {
	url url.URL

	headers map[string]string
	wrap    http.RoundTripper
}

func (t *customHeadersTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, val := range t.headers {
		req.Header.Add(key, val)
	}

	newURL := t.url
	newURL.Path += req.URL.Path
	newURL.RawQuery = req.URL.RawQuery

	req.URL = &newURL

	By(req.URL.String())

	return t.wrap.RoundTrip(req)
}

const (
	IamPluginPath     = "flant"
	IamAuthPluginPath = "auth/flant"
)

func NewIamVaultClient(token string) *http.Client {
	return NewVaultClient(GetRootVaultUrl()+"/v1/", token, IamPluginPath)
}

func NewConfiguredIamVaultClient() *http.Client {
	token := GetRootRootToken()
	return NewIamVaultClient(token)
}

func NewIamAuthVaultClient(token string) *http.Client {
	return NewVaultClient(GetAuthVaultUrl()+"/v1/", token, IamAuthPluginPath)
}

func NewConfiguredIamAuthVaultClient() *http.Client {
	token := GetAuthRootToken()
	return NewIamAuthVaultClient(token)
}

func NewVaultClient(baseURL string, token string, pluginPath string) *http.Client {
	pluginURL, err := url.Parse(baseURL + pluginPath)
	if err != nil {
		panic(err)
	}

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}

	if len(token) > 0 {
		headers["X-Vault-Token"] = token
	}

	tr := &customHeadersTransport{
		url:     *pluginURL,
		headers: headers,
		wrap: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // We do not want to test tls, only logic
			},
			IdleConnTimeout:       5 * time.Minute,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext:           dialer.DialContext,
			Dial:                  dialer.Dial,
		},
	}

	return &http.Client{Transport: tr}
}

func HttpClientWithoutInsequireVerifing() *http.Client {
	client := http.DefaultClient
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		}}
	return client
}

func GetRootRootToken() string {
	token := os.Getenv("ROOT_VAULT_TOKEN")
	if token == "" {
		panic("ROOT_VAULT_TOKEN is empty, need valid token to access vault")
	}
	return token
}

func GetAuthRootToken() string {
	token := os.Getenv("AUTH_VAULT_TOKEN")
	if token == "" {
		panic("AUTH_VAULT_TOKEN is empty, need valid token to access vault")
	}
	return token
}

func GetRootVaultUrl() string {
	u := os.Getenv("ROOT_VAULT_URL")
	if u == "" {
		panic("ROOT_VAULT_URL is empty, need valid URL to access vault")
	}
	return u
}

func GetAuthVaultUrl() string {
	u := os.Getenv("AUTH_VAULT_URL")
	if u == "" {
		panic("AUTH_VAULT_URL is empty, need valid URL to access vault")
	}
	return u
}

func WaitDataReachFlantAuthPlugin(maxAttempts int, vaultUrl string) error {
	rootIamClient := NewConfiguredIamVaultClient()
	tenant := specs.CreateRandomTenant(NewTenantAPI(rootIamClient))
	user := specs.CreateRandomUser(NewUserAPI(rootIamClient), tenant.UUID)
	_, multipassJWT := specs.CreateUserMultipass(NewUserMultipassAPI(rootIamClient),
		user, "test", 100*time.Second, 1000*time.Second, []string{"ssh.open"})
	f := func() error { return TryLoginByMultipassJWTToVault(multipassJWT, vaultUrl) }
	return tests.Repeat(f, maxAttempts)
}

func TryLoginByMultipassJWTToVault(multipassJWT string, vaultUrl string) error {
	url := vaultUrl + "/v1/auth/flant/login"
	payload := map[string]interface{}{
		"method": "multipass",
		"jwt":    multipassJWT,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return nil
	}
	client := HttpClientWithoutInsequireVerifing()
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("wrong response status:%d", resp.StatusCode)
	}
	return nil
}
