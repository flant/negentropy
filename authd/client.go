package authd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"

	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/authd/pkg/util"
	"github.com/flant/negentropy/authd/pkg/util/exponential"
	"github.com/flant/negentropy/authd/pkg/vault"
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
	c.m.RLock()
	resCl.SetToken(c.token)
	c.m.RUnlock()
	return resCl, nil
}

var refreshTime = time.Minute

func (c *Client) StartTokenRefresher(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	c.refresherDone = make(chan struct{})
	c.refresherCtx, c.refresherCancel = context.WithCancel(ctx)

	go func() {
		defer close(c.refresherDone)
		ticker := time.NewTicker(refreshTime)
		for {
			select {
			case <-ticker.C:
				tokenRefresher := &util.PostponedRetryLoop{
					Handler: func(ctx2 context.Context) error {
						c.m.Lock()
						defer c.m.Unlock()
						cfg := api.DefaultConfig()
						cfg.Address = c.server
						cl, err := api.NewClient(cfg)
						if err != nil {
							return err
						}
						cl.SetToken(c.token)
						TenMinutes := 60 * 10
						auth, err := cl.Auth().Token().RenewSelf(TenMinutes)
						if err != nil {
							return err
						}
						if auth == nil {
							return fmt.Errorf("self_renew token:empty response")
						}
						c.token = auth.Auth.ClientToken
						return nil
					},
					Backoff: exponential.NewBackoff(time.Second, time.Minute, 2.0),
				}
				tokenRefresher.RunLoop(c.refresherCtx) // nolint:errcheck

			case <-c.refresherCtx.Done():
				return
			}
		}
	}()

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
