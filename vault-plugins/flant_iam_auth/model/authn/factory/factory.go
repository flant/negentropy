package factory

import (
	"context"
	"fmt"
	"sync"

	hcjwt "github.com/hashicorp/cap/jwt"
	"github.com/hashicorp/go-hclog"

	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn/jwt"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/authn/multipass"
	ext_repo "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	njwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
)

type AuthenticatorFactory struct {
	logger        hclog.Logger
	jwtController *njwt.Controller

	l          sync.RWMutex
	validators map[string]*hcjwt.Validator
}

func NewAuthenticatorFactory(jwtController *njwt.Controller, logger hclog.Logger) *AuthenticatorFactory {
	return &AuthenticatorFactory{
		logger:        logger,
		jwtController: jwtController,
		validators:    map[string]*hcjwt.Validator{},
	}
}

func (f *AuthenticatorFactory) Reset() {
	f.l.Lock()
	defer f.l.Unlock()

	f.validators = map[string]*hcjwt.Validator{}
}

func (f *AuthenticatorFactory) GetAuthenticator(ctx context.Context, method *model.AuthMethod, tnx *io.MemoryStoreTxn) (authn.Authenticator, *model.AuthSource, error) {
	switch method.MethodType {
	case model.MethodTypeJWT:
		return f.jwt(ctx, method, tnx)

	case model.MethodTypeMultipass:
		return f.multipass(ctx, method, tnx)
	}

	f.logger.Warn(fmt.Sprintf("unsupported auth method %s", method.MethodType))
	return nil, nil, fmt.Errorf("unsupported auth method")
}

func (f *AuthenticatorFactory) getAuthSource(method *model.AuthMethod, tnx *io.MemoryStoreTxn) (*model.AuthSource, error) {
	repo := ext_repo.NewAuthSourceRepo(tnx)
	authSource, err := repo.Get(method.Source)
	if err != nil {
		return nil, err
	}
	if authSource == nil {
		return nil, fmt.Errorf("not found auth method")
	}

	return authSource, nil
}

func (f *AuthenticatorFactory) jwt(ctx context.Context, method *model.AuthMethod, tnx *io.MemoryStoreTxn) (authn.Authenticator, *model.AuthSource, error) {
	authSource, err := f.getAuthSource(method, tnx)
	if err != nil {
		return nil, nil, err
	}

	jwtValidator, err := f.jwtValidator(ctx, method.Name, authSource)
	if err != nil {
		return nil, nil, err
	}

	return &jwt.Authenticator{
		AuthMethod:   method,
		Logger:       f.logger.Named("JWTAutheNticator"),
		AuthSource:   authSource,
		JwtValidator: jwtValidator,
	}, authSource, nil
}

func (f *AuthenticatorFactory) multipass(ctx context.Context, method *model.AuthMethod, tnx *io.MemoryStoreTxn) (authn.Authenticator, *model.AuthSource, error) {
	f.logger.Debug("It is multipass. Check jwt is enabled")

	enabled, err := f.jwtController.IsEnabled(tnx)
	if err != nil {
		return nil, nil, err
	}
	if !enabled {
		f.logger.Warn("jwt is not enabled. not use multipass login")
		return nil, nil, fmt.Errorf("jwt is not enabled. not use multipass login")
	}

	f.logger.Debug("Jwt is enabled. Get jwt public keys")

	keys, err := f.jwtController.JWKS(tnx)
	if err != nil {
		return nil, nil, err
	}

	f.logger.Debug(fmt.Sprintf("Got jwt keys. %s Get jwt config", keys))

	jwtConf, err := f.jwtController.GetConfig(tnx)
	if err != nil {
		return nil, nil, err
	}

	f.logger.Debug("Got jwt config")

	authSource := model.GetMultipassSourceForLogin(jwtConf, keys)
	jwtValidator, err := f.jwtValidator(ctx, method.Name, authSource)
	if err != nil {
		return nil, nil, err
	}

	loggerForAuth := f.logger.Named("MultipassAutheNticator")

	authenticator := &multipass.Authenticator{
		AuthSource:   authSource,
		AuthMethod:   method,
		JwtValidator: jwtValidator,
		Logger:       loggerForAuth,
		MultipassService: &usecase.Multipass{
			JwtController:    f.jwtController,
			MultipassRepo:    iam_repo.NewMultipassRepository(tnx),
			GenMultipassRepo: model.NewMultipassGenerationNumberRepository(tnx),
			Logger:           loggerForAuth,
		},
	}

	return authenticator, authSource, nil
}

// jwtValidator returns a new JWT validator based on the provided config.
func (f *AuthenticatorFactory) jwtValidator(ctx context.Context, methodName string, config *model.AuthSource) (*hcjwt.Validator, error) {
	f.l.Lock()
	defer f.l.Unlock()

	if v, ok := f.validators[methodName]; ok {
		return v, nil
	}

	var err error
	var keySet hcjwt.KeySet

	// Configure the key set for the validator
	switch config.AuthType() {
	case model.AuthSourceJWKS:
		keySet, err = hcjwt.NewJSONWebKeySet(ctx, config.JWKSURL, config.JWKSCAPEM)
	case model.AuthSourceStaticKeys:
		keySet, err = hcjwt.NewStaticKeySet(config.ParsedJWTPubKeys)
	case model.AuthSourceOIDCDiscovery:
		keySet, err = hcjwt.NewOIDCDiscoveryKeySet(ctx, config.OIDCDiscoveryURL, config.OIDCDiscoveryCAPEM)
	default:
		return nil, fmt.Errorf("unsupported config type")
	}

	if err != nil {
		return nil, fmt.Errorf("keyset configuration error: %w", err)
	}

	validator, err := hcjwt.NewValidator(keySet)
	if err != nil {
		return nil, fmt.Errorf("JWT validator configuration error: %w", err)
	}

	// not cache multipass validator
	// TODO flush cache when update JWKS
	if config.UUID != model.MultipassSourceUUID {
		f.validators[methodName] = validator
	}

	return validator, nil
}
