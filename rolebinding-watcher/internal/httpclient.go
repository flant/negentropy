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
	Client *http.Client
	URL    string
}

func (c *HTTPClient) ProceedUserEffectiveRole(userEffectiveRoles pkg.UserEffectiveRoles) error {
	data, err := json.Marshal(userEffectiveRoles)
	if err != nil {
		return err
	}
	operation := func() error {
		resp, err := c.Client.Post(c.URL, "application/json", bytes.NewReader(data))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("wrong response: %d", resp.StatusCode)
		}
		return nil
	}
	err = backoff.Retry(operation, sharedio.ThirtySecondsBackoff())
	if err != nil {
		return err
	}
	return nil
}

func NewHTTPClient(url string) *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{},
		URL:    url,
	}
}
