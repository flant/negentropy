package vault

import (
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/api"
)

type RedirectSaverClient struct {
	*api.Client
	server        string
	initialScheme string
}

func NewRedirectSaverClient(cfg *api.Config) (*RedirectSaverClient, error) {
	res := &RedirectSaverClient{
		server:        cfg.Address,
		initialScheme: DetectScheme(cfg.Address),
	}
	if cfg.HttpClient == nil {
		cfg.HttpClient = cleanhttp.DefaultClient()
	}
	// CheckRedirect forbid redirects — the retryablehttp.Client used in api.Client
	// expects this behavior (see api.DefaultConfig()).
	cfg.HttpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Ensure new server has scheme prefix and save as a new target.
		// Note: 'https' to 'http' downgrade is not allowed by vault client,
		// but it should not be fixed here.
		res.server = EnsureScheme(req.URL.Host, res.initialScheme)
		return http.ErrUseLastResponse
	}

	cl, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	res.Client = cl
	return res, nil
}

func (c *RedirectSaverClient) GetTargetServer() string {
	return c.server
}
