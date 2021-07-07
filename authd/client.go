package authd

import (
	"context"
	"fmt"
	"github.com/flant/negentropy/authd/pkg/vault"
	"os"
	"sync"
	"time"

	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/authd/pkg/util"
	"github.com/hashicorp/vault/api"
)

/**
Example:

// Open session with default auth server. authd returns specific
// server after redirects and a session token to work with vault.
authdClient := authd.NewAuthdClient("/run/authd/authd.sock")
err := authdClient.OpenVaultSession(v1.NewDefaultLoginToAuthServer())
if err != nil {...}
// Start session token refresh in background.
go authdClient.StartTokenRefresher(context.Background())
defer authdClient.StopTokenRefresher()

vaultClient := authdClient.NewVaultClient()

// Do some vault stuff...
vaultClient.SSH().SignKey(...)


*/

type Client struct {
	SocketPath string

	token  string
	server string

	m               sync.RWMutex // mutex to sync NewVaultClient and token refresher.
	refresher       *util.PostponedRetryLoop
	refresherCtx    context.Context
	refresherCancel context.CancelFunc
	refresherDone   chan struct{}
}

func NewAuthdClient(socketPath string) *Client {
	return &Client{
		SocketPath: socketPath,
	}
}

func (c *Client) NewVaultClient() (*api.Client, error) {
	cfg := api.DefaultConfig()
	cfg.Address = c.server

	resCl, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	resCl.SetToken(c.token)
	return resCl, nil
}

// TODO refresher for client session token.
func (c *Client) StartTokenRefresher(ctx context.Context) {
	c.refresherDone = make(chan struct{})
	defer close(c.refresherDone)
	for {
		if ctx.Err() != nil {
			return
		}
		time.Sleep(10 * time.Second)
	}
}

func (c *Client) StopTokenRefresher() {
	if c.refresherCancel != nil {
		c.refresherCancel()
	}
	<-c.refresherDone
}

func (c *Client) checkSocket() error {
	exists, err := util.FileExists(c.SocketPath)
	if err != nil {
		return fmt.Errorf("check authd socket '%s': %s", c.SocketPath, err)
	}
	if !exists {
		return fmt.Errorf("authd socket '%s' is not exists", c.SocketPath)
	}
	return nil
}

// OpenVaultSession asks authd to return new vault token and a specific vault server.
// TODO add context argument.
func (c *Client) OpenVaultSession(loginRequest *v1.LoginRequest) error {
	err := c.checkSocket()
	if err != nil {
		return err
	}
	// Use vault client to talk to authd over unix socket.
	cfg := &api.Config{
		Address: "unix://" + c.SocketPath,
	}
	cl, err := api.NewClient(cfg)
	if err != nil {
		return err
	}
	// FIXME: authd login path is hardcoded in the client library.
	req := cl.NewRequest("POST", fmt.Sprintf("/v1/login/%s", loginRequest.ServerType))
	if err := req.SetJSONBody(loginRequest); err != nil {
		return err
	}

	if os.Getenv("AUTHD_DEBUG") == "yes" {
		req.Params.Set("debug", "yes")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	resp, err := cl.RawRequestWithContext(ctx, req)
	if err != nil {
		return fmt.Errorf("raw request: %v", err)
	}
	defer resp.Body.Close()

	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return err
	}

	c.server = vault.SecretDataGetString(secret, "server")
	c.token = secret.Auth.ClientToken
	return nil
}
