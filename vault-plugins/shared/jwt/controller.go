package jwt

import (
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"time"

	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/backend"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
	"github.com/hashicorp/go-hclog"
)

type Controller struct {
	deps *usecase.Depends
	backend *backend.Backend
}

func NewJwtController(storage *sharedio.MemoryStore, idGetter func() (string, error), logger hclog.Logger, now func() time.Time) *Controller {
	deps := usecase.NewDeps(idGetter, logger, now)
	b := backend.NewBackend(storage, deps)
	return &Controller{
		deps:    deps,
		backend: b,
	}
}

func (c *Controller) IssueMultipass(tnx *sharedio.MemoryStoreTxn, options *usecase.PrimaryTokenOptions) (string, error) {
	iss, err := c.issuer(tnx)
	if err != nil {
		return "", nil
	}
	return iss.PrimaryToken(options)
}

func (c *Controller) IssuePayloadAsJwt(tnx *sharedio.MemoryStoreTxn, payload map[string]interface{}, options *usecase.TokenOptions) (string, error) {
	iss, err := c.issuer(tnx)
	if err != nil {
		return "", nil
	}

	return iss.Token(payload, options)
}

func (c *Controller) IsEnabled(db *sharedio.MemoryStoreTxn) (bool, error) {
	stateRepo, err := c.deps.StateRepo(db)
	if err != nil {
		return false, err
	}

	enabled, err := stateRepo.IsEnabled()
	if err != nil {
		return false, err
	}

	return enabled, nil
}

func (c *Controller) OnPeriodic(tnx *sharedio.MemoryStoreTxn) error {
	enabled, err := c.IsEnabled(tnx)
	if err != nil {
		return err
	}

	if !enabled {
		return nil
	}

	keyPair, err := c.deps.KeyPairsService(tnx)
	if err != nil {
		return nil
	}

	err = keyPair.RunPeriodicalRotateKeys()
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) GetConfig(tnx *sharedio.MemoryStoreTxn) (*model.Config, error) {
	return c.deps.ConfigRepo(tnx).Get()
}

func (c *Controller) ApiPaths() []*framework.Path {
	return []*framework.Path{
		backend.PathEnable(c.backend),
		backend.PathDisable(c.backend),
		backend.PathConfigure(c.backend),
		backend.PathJWKS(c.backend),
		backend.PathRotateKey(c.backend),
	}
}

func (c *Controller) issuer(tnx *sharedio.MemoryStoreTxn) (*usecase.TokenIssuer, error) {
	enabled, err := c.IsEnabled(tnx)
	if err != nil {
		return nil, err
	}

	if !enabled {
		return nil, fmt.Errorf("jwt is disabled")
	}

	config, err := c.GetConfig(tnx)
	if err != nil {
		return nil, err
	}

	stateRepo, err := c.deps.StateRepo(tnx)
	if err != nil {
		return nil, err
	}

	keyPair, err := stateRepo.GetKeyPair()
	if err != nil {
		return nil, err
	}

	if keyPair == nil || len(keyPair.PrivateKeys.Keys) == 0 {
		return nil, fmt.Errorf("can not found valid private key")
	}

	firtsKey := keyPair.PrivateKeys.Keys[0].JSONWebKey

	return usecase.NewTokenIssuer(config, &firtsKey, c.deps.Now), nil
}

type MultipassIssFn func(options *usecase.PrimaryTokenOptions) (string, error)

func CreateIssueMultipassFunc(c *Controller, tnx *sharedio.MemoryStoreTxn) MultipassIssFn {
	return func(options *usecase.PrimaryTokenOptions) (string, error) {
		return c.IssueMultipass(tnx, options)
	}
}
