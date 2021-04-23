package jwt

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"gopkg.in/square/go-jose.v2"
)

// Factory is used by framework
func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := backend()
	if err := b.Setup(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

// Simple backend for test purposes (treat it like an example)
type jwtAuthBackend struct {
	*framework.Backend

	tokenController *TokenController
}

func backend() *jwtAuthBackend {
	b := new(jwtAuthBackend)
	b.tokenController = NewTokenController()

	b.Backend = &framework.Backend{
		BackendType:  logical.TypeCredential,
		Help:         backendHelp,
		PathsSpecial: &logical.Paths{},
		Paths: framework.PathAppend(
			[]*framework.Path{
				PathEnable(b.tokenController),
				PathDisable(b.tokenController),
				PathConfigure(b.tokenController),
				PathJWKS(b.tokenController),
				PathRotateKey(b.tokenController),
			},
		),
	}

	return b
}

type TokenController struct {
	// mu sync.RWMutex
}

func NewTokenController() *TokenController {
	return &TokenController{}
}

func (b *TokenController) rotateKeys(ctx context.Context, logger hclog.Logger, req logical.Request) {
	entry, err := req.Storage.Get(ctx, "jwt/enable")
	if err != nil {
		logger.Warn(err.Error())
		return
	}

	var enabled bool
	if entry != nil {
		err = entry.DecodeJSON(&enabled)
		if err != nil {
			logger.Warn(err.Error())
			return
		}
	}

	if !enabled {
		return
	}

	entry, err = req.Storage.Get(ctx, "jwt/jwks")
	if err != nil {
		logger.Warn(err.Error())
		return
	}

	keys := make([]byte, 0)
	if entry != nil {
		keys = entry.Value
	}

	keysSet := jose.JSONWebKeySet{}
	if len(keys) > 0 {
		if err := json.Unmarshal(keys, &keysSet); err != nil {
			logger.Warn(err.Error())
			return
		}
	} else {
		logger.Warn("cannot find keys in the store")
		return
	}
}

const (
	backendHelp = `
The JWT backend allows to generate JWT tokens.
`
)
