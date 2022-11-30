package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cenkalti/backoff"

	"github.com/flant/negentropy/rolebinding-watcher/pkg"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
)

type HTTPClient struct {
	Client      *http.Client
	URL         string
	HeaderName  string
	HeaderValue string
}

func (c *HTTPClient) ProceedUserEffectiveRole(userEffectiveRoles pkg.UserEffectiveRoles) error {
	data, err := json.Marshal(userEffectiveRoles)
	if err != nil {
		return err
	}
	operation := func() error {
		resp, err := c.Post(c.URL, "application/json", bytes.NewReader(data))
		if err != nil {
			return err
		}
		defer resp.Body.Close() // nolint: errcheck
		if resp.StatusCode != 200 {
			return fmt.Errorf("wrong response: %d", resp.StatusCode)
		}
		return nil
	}
	err = backoff.Retry(operation, sharedio.ThirtySecondsBackoff())
	return err
}

func (c *HTTPClient) Post(url string, contentType string, body *bytes.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	if c.HeaderName != "" && c.HeaderValue != "" {
		req.Header.Set(c.HeaderName, c.HeaderValue)
	}
	return c.Client.Do(req)
}

func NewHTTPClient(url string, headerName string, headerValue string) *HTTPClient {
	return &HTTPClient{
		Client:      &http.Client{},
		URL:         url,
		HeaderName:  headerName,
		HeaderValue: headerValue,
	}
}
