package lib

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
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
	defaultBaseURL    = "http://127.0.0.1:8200/v1/"
	IamPluginPath     = "flant_iam"
	IamAuthPluginPath = "auth/flant_iam_auth"
)

func NewIamVaultClient(token string) *http.Client {
	return NewVaultClient(GetRootVaultBaseUrl(), token, IamPluginPath)
}

func NewConfiguredIamVaultClient() *http.Client {
	token := GetSecondRootToken()
	return NewVaultClient(GetRootVaultBaseUrl(), token, IamPluginPath)
}

func NewIamAuthVaultClient(token string) *http.Client {
	return NewVaultClient(GetAuthVaultBaseUrl(), token, IamAuthPluginPath)
}

func NewConfiguredIamAuthVaultClient() *http.Client {
	token := GetSecondRootToken()
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

func GetSecondRootToken() string {
	token := os.Getenv("TEST_VAULT_SECOND_TOKEN")
	if token == "" {
		panic("TEST_VAULT_SECOND_TOKEN is empty, need valid token to access vault")
	}
	return token
}

func GetRootVaultBaseUrl() string {
	u := os.Getenv("ROOT_VAULT_BASE_URL")
	if u == "" {
		return defaultBaseURL
	}
	return u
}

func GetAuthVaultBaseUrl() string {
	u := os.Getenv("AUTH_VAULT_BASE_URL")
	if u == "" {
		return defaultBaseURL
	}
	return u
}
