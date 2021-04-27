package vault

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/api"
)

type Client struct {
	Server string
}

const LoginURI = "/login"
const JWTLoginURL = "/v1/auth/myjwt/login"
const ObtainJWTURL = "/v1/identity/oidc/token/myrole"

func (c *Client) LoginWithJWT(jwt string) (*api.Secret, error) {
	// TODO create specific config, detect Redirect.
	cfg := &api.Config{
		Address: "http://127.0.0.1:8200",
	}
	cl, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	req := cl.NewRequest("POST", JWTLoginURL)
	opts := map[string]interface{}{
		"role": "authd",
		"jwt":  jwt,
	}
	if err := req.SetJSONBody(opts); err != nil {
		return nil, err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	resp, err := cl.RawRequestWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vault request: %v", err)
	}
	defer resp.Body.Close()

	return api.ParseSecret(resp.Body)
}

func (c *Client) CheckPendingLogin(token string) (interface{}, error) {
	return nil, nil
}

// RefreshJWT opens a new session with vault and issue a new JWT.
func (c *Client) RefreshJWT(jwt string) (string, error) {
	secret, err := c.LoginWithJWT(jwt)
	if err != nil {
		return "", err
	}

	cfg := &api.Config{
		Address: "http://127.0.0.1:8200",
	}
	cl, err := api.NewClient(cfg)
	if err != nil {
		return "", err
	}
	cl.SetToken(secret.Auth.ClientToken)

	req := cl.NewRequest("POST", ObtainJWTURL)
	opts := map[string]interface{}{
		"role": "authd",
		"jwt":  jwt,
	}
	if err := req.SetJSONBody(opts); err != nil {
		return "", err
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	resp, err := cl.RawRequestWithContext(ctx, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	oidcSecret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return "", err
	}

	return oidcSecret.Data["token"].(string), nil
}

func (c *Client) newHttpClient() *http.Client {
	return http.DefaultClient
}
