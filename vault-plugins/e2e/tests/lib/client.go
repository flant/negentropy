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
	baseURL    = "http://127.0.0.1:8200/v1/"
	pluginPath = "flant_iam"

	RootToken = "root"
)

func GetVaultClient(token string) *http.Client {
	pluginURL, err := url.Parse(baseURL + pluginPath)
	if err != nil {
		panic(err)
	}

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	tr := &customHeadersTransport{
		url: *pluginURL,
		headers: map[string]string{
			"Content-Type":  "application/json",
			"Accept":        "application/json",
			"X-Vault-Token": token,
		},
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
	return os.Getenv("TEST_VAULT_SECOND_TOKEN")
}
