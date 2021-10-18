package jwt

import (
	"crypto"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"

	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/backend"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
)

type Controller struct {
	deps    *usecase.Depends
	backend *backend.Backend
}

func NewJwtController(storage *sharedio.MemoryStore, idGetter func() (string, error), parentLogger hclog.Logger, now func() time.Time) *Controller {
	deps := usecase.NewDeps(idGetter, parentLogger.Named("JWT.Controller"), now)
	b := backend.NewBackend(storage, deps)
	return &Controller{
		deps:    deps,
		backend: b,
	}
}

func (c *Controller) IssueMultipass(txn *sharedio.MemoryStoreTxn, options *usecase.PrimaryTokenOptions) (string, error) {
	iss, err := c.issuer(txn)
	if err != nil {
		return "", err
	}
	return iss.PrimaryToken(options)
}

func (c *Controller) IssuePayloadAsJwt(txn *sharedio.MemoryStoreTxn, payload map[string]interface{}, options *usecase.TokenOptions) (string, error) {
	iss, err := c.issuer(txn)
	if err != nil {
		return "", err
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

func (c *Controller) JWKS(db *sharedio.MemoryStoreTxn) ([]crypto.PublicKey, error) {
	jwksRepo, err := c.deps.JwksRepo(db)
	if err != nil {
		return nil, err
	}

	set, err := jwksRepo.GetSet()
	if err != nil {
		return nil, err
	}

	if len(set) == 0 {
		return nil, fmt.Errorf("Empty key set")
	}

	keys := make([]crypto.PublicKey, 0)
	for _, s := range set {
		keys = append(keys, s.Key)
	}

	return keys, nil
}

func (c *Controller) CalcMultipassJTI(tokenNumber int64, salt string) string {
	return usecase.TokenJTI{
		Generation: tokenNumber,
		SecretSalt: salt,
	}.Hash()
}

func (c *Controller) OnPeriodical(txn *sharedio.MemoryStoreTxn) error {
	enabled, err := c.IsEnabled(txn)
	if err != nil {
		return err
	}

	if !enabled {
		c.deps.Logger.Warn("jwks was not rotated bacause jwt is disabled")
		return nil
	}

	keyPair, err := c.deps.KeyPairsService(txn)
	if err != nil {
		return nil
	}

	err = keyPair.RunPeriodicalRotateKeys()
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) GetConfig(txn *sharedio.MemoryStoreTxn) (*model.Config, error) {
	return c.deps.ConfigRepo(txn).Get()
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

func (c *Controller) issuer(txn *sharedio.MemoryStoreTxn) (*usecase.TokenIssuer, error) {
	enabled, err := c.IsEnabled(txn)
	if err != nil {
		return nil, err
	}

	if !enabled {
		return nil, fmt.Errorf("jwt is disabled")
	}

	config, err := c.GetConfig(txn)
	if err != nil {
		return nil, err
	}

	stateRepo, err := c.deps.StateRepo(txn)
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

func CreateIssueMultipassFunc(c *Controller, txn *sharedio.MemoryStoreTxn) MultipassIssFn {
	return func(options *usecase.PrimaryTokenOptions) (string, error) {
		return c.IssueMultipass(txn, options)
	}
}
