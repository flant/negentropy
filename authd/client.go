package authd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/authd/pkg/util"
	"github.com/hashicorp/vault/api"
)

/**
Example:

authdClient := new(authd.Client)
vaultClient, err := authdClient.Login(v1.NewDefaultLoginToAuthServer())
if err != nil {...}
vaultClient.SSH().SignKey(...)

go tokenRefresher(vaultClient).Start()

*/

type Client struct {
	SocketPath string
}

func (c *Client) newHttpClient() (*http.Client, error) {
	exists, err := util.FileExists(c.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("check authd socket '%s': %s", c.SocketPath, err)
	}
	if !exists {
		return nil, fmt.Errorf("authd socket '%s' is not exists", c.SocketPath)
	}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", c.SocketPath)
			},
		},
	}, nil
}

// Login asks authd to return new vault token and returns new vault client
// with specific server and token.
func (c *Client) Login(loginRequest *v1.LoginRequest) (*api.Client, error) {
	socketClient, err := c.newHttpClient()
	if err != nil {
		return nil, err
	}

	cfg := &api.Config{
		Address:    "http://unix",
		HttpClient: socketClient,
	}
	cl, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	req := cl.NewRequest("POST", fmt.Sprintf("/v1/login/%s", loginRequest.ServerType))
	if err := req.SetJSONBody(loginRequest); err != nil {
		return nil, err
	}

	if os.Getenv("AUTHD_DEBUG") == "yes" {
		req.Params.Set("debug", "yes")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	resp, err := cl.RawRequestWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("raw request: %v", err)
	}
	defer resp.Body.Close()

	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return nil, err
	}

	bt, _ := json.Marshal(secret)
	fmt.Printf("secret: %s\n", string(bt))

	// TODO unmarshal Data to struct.
	// TODO do something with schema.
	resCfg := &api.Config{
		Address: "http://" + secret.Data["server"].(string),
	}
	resCl, err := api.NewClient(resCfg)
	if err != nil {
		return nil, err
	}
	resCl.SetToken(secret.Auth.ClientToken)
	return resCl, nil
}

// TODO create refresher for session token.
/**

 */
