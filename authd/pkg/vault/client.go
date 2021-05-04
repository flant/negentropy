package vault

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

type Client struct {
	Server string
	Scheme string
}

const JWTLoginURL = "/v1/auth/myjwt/login"
const ObtainJWTURL = "/v1/identity/oidc/token/myrole"

var DefaultScheme = "https"

func NewClient(addr string) *Client {
	return &Client{
		Server: addr,
		Scheme: DetectScheme(addr),
	}
}

// LoginWithJWT use JWT to auth in Vault and get session token.
//
// Also it follows redirects and return last used server in Data map of api.Secret object.
func (c *Client) LoginWithJWT(ctx context.Context, jwt string) (*api.Secret, error) {
	cfg := api.DefaultConfig()
	cfg.Address = c.PrepareServerAddr(c.Server)
	cl, err := NewRedirectSaverClient(cfg)
	if err != nil {
		return nil, err
	}

	// FIXME(far future): NewRequest do an uninterrupted SRV lookup if Port is not specified.
	req := cl.NewRequest("POST", JWTLoginURL)
	opts := map[string]interface{}{
		"role": "authd",
		"jwt":  jwt,
	}
	if err := req.SetJSONBody(opts); err != nil {
		return nil, err
	}

	resp, err := cl.RawRequestWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vault request: %v", err)
	}
	defer resp.Body.Close()

	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		// TODO Think about a sanity response for client if Body was empty.
		return nil, errors.New("Login return empty secret. May be redirect misconfiguration or network problems.")
	}
	// Return last used server in Data map.
	logrus.Debugf("server after login: '%s', prepared: '%s'", cl.GetTargetServer(), c.PrepareServerAddr(cl.GetTargetServer()))
	SecretDataSetString(secret, "server", c.PrepareServerAddr(cl.GetTargetServer()))
	return secret, nil
}

func (c *Client) CheckPendingLogin(token string) (interface{}, error) {
	return nil, nil
}

// RefreshJWT opens a new session with vault using current JWT
// and obtains a new, "refreshed" JWT.
func (c *Client) RefreshJWT(ctx context.Context, jwt string) (string, error) {
	secret, err := c.LoginWithJWT(ctx, jwt)
	if err != nil {
		return "", err
	}

	server := SecretDataGetString(secret, "server", c.Server)
	token := secret.Auth.ClientToken

	logrus.Debugf("server: '%s' token: '%s'", server, token)

	cfg := api.DefaultConfig()
	cfg.Address = c.PrepareServerAddr(server)
	logrus.Debugf("cfg Address: '%s'", cfg.Address)

	cl, err := api.NewClient(cfg)
	if err != nil {
		return "", err
	}
	cl.SetToken(token)

	req := cl.NewRequest("GET", ObtainJWTURL)

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

// PrepareServerAddr adds schema to a server address.
func (c *Client) PrepareServerAddr(server string) string {
	if strings.HasPrefix(server, "unix://") || strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://") {
		return server
	}
	server = strings.TrimSuffix(server, "://")

	return c.Scheme + "://" + server
}

func DetectScheme(addr string) string {
	switch true {
	case strings.HasPrefix(addr, "unix://") || strings.HasPrefix(addr, "http://"):
		return "http"
	case strings.HasPrefix(addr, "https://"):
		return "https"
	}
	return DefaultScheme
}

func EnsureScheme(addr string, scheme string) string {
	parts := strings.SplitN(addr, "://", 2)
	if len(parts) == 2 {
		// TODO: we can't rewrite scheme, vault client returns error on downgrade.
		// Do not allow downgrade.
		//if scheme == "https" && parts[0] == "http" {
		//	return scheme+"://"+parts[1]
		//}
		return addr
	}
	return scheme + "://" + addr
}
