package vault

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"

	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/authd/pkg/client_error"
	jwt2 "github.com/flant/negentropy/authd/pkg/jwt"
)

type Client struct {
	Server        string
	Scheme        string
	LoginEndpoint string
}

const (
	DefaultLoginEndpoint = "/v1/auth/flant_iam_auth/login"
	ObtainJWTURL         = "/v1/auth/flant_iam_auth/issue/multipass_jwt/"
)

var DefaultScheme = "https"

func NewClient(addr string) *Client {
	cl := &Client{
		Server:        addr,
		Scheme:        DetectScheme(addr),
		LoginEndpoint: os.Getenv("AUTHD_LOGIN_ENDPOINT"),
	}

	if cl.LoginEndpoint == "" {
		cl.LoginEndpoint = DefaultLoginEndpoint
	}
	return cl
}

// LoginWithJWT use JWT to auth in Vault and get session token.
//
// Also it follows redirects and return last used server in Data map of api.Secret object.
func (c *Client) LoginWithJWTAndClaims(ctx context.Context, jwt string, claimedRoles []v1.RoleWithClaim) (*api.Secret, error) {
	cfg := api.DefaultConfig()
	cfg.Address = c.PrepareServerAddr(c.Server)
	cl, err := NewRedirectSaverClient(cfg)
	if err != nil {
		return nil, err
	}

	// FIXME(far future): NewRequest do an uninterrupted SRV lookup if Port is not specified.
	// TODO: make settings for login.
	req := cl.NewRequest("POST", c.LoginEndpoint)
	opts := map[string]interface{}{
		"method": "multipass",
		"jwt":    jwt,
	}
	if len(claimedRoles) > 0 {
		opts["roles"] = claimedRoles
	}

	if err := req.SetJSONBody(opts); err != nil {
		return nil, err
	}

	resp, err := cl.RawRequestWithContext(ctx, req)
	if err != nil {
		if resp.StatusCode == http.StatusForbidden {
			return nil, client_error.NewHTTPError(err, http.StatusForbidden, []string{"vault returns 403 status"})
		}
		return nil, fmt.Errorf("%w: vault reponse: %v", err, resp.Body)
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

// login use JWT to auth in Vault and get session token.
func (c *Client) login(ctx context.Context, jwt string) (*api.Secret, error) {
	return c.LoginWithJWTAndClaims(ctx, jwt, nil)
}

// RefreshJWT opens a new session with vault using current JWT
// and obtains a new, "refreshed" JWT.
func (c *Client) RefreshJWT(ctx context.Context, jwt string) (string, error) {
	secret, err := c.login(ctx, jwt)
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

	t, err := jwt2.ParseToken(jwt)
	if err != nil {
		return "", err
	}
	var uuid string
	var ok bool
	if uuid, ok = t.Payload["sub"].(string); !ok {
		return "", fmt.Errorf("wrong payload, need key='sub', got:%#v", t.Payload)
	}
	req := cl.NewRequest("PUT", ObtainJWTURL+uuid)

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
