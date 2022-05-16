package factory

import (
	"context"
	"fmt"
	"sync"

	hcjwt "github.com/hashicorp/cap/jwt"
	"github.com/hashicorp/go-hclog"

	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase"
	authn2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/accesstoken"
	jwt2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/jwt"
	multipass2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/multipass"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/serviceaccountpass"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	njwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
)

type AuthenticatorFactory struct {
	logger        hclog.Logger
	jwtController *njwt.Controller

	l          sync.RWMutex
	validators map[string]jwt2.JwtValidator
}

func NewAuthenticatorFactory(jwtController *njwt.Controller, parentLogger hclog.Logger) *AuthenticatorFactory {
	return &AuthenticatorFactory{
		logger:        parentLogger.Named("Login"),
		jwtController: jwtController,
		validators:    map[string]jwt2.JwtValidator{},
	}
}

func (f *AuthenticatorFactory) Reset() {
	f.l.Lock()
	defer f.l.Unlock()

	f.validators = map[string]jwt2.JwtValidator{}
}

func (f *AuthenticatorFactory) GetAuthenticator(ctx context.Context, method *model.AuthMethod, txn *io.MemoryStoreTxn) (authn2.Authenticator, *model.AuthSource, error) {
	f.logger.Debug(fmt.Sprintf("Auth method: %s", method.MethodType))

	switch method.MethodType {
	case model.MethodTypeJWT:
		return f.jwt(ctx, method, txn)

	case model.MethodTypeMultipass:
		return f.multipass(ctx, method, txn)

	case model.MethodTypeAccessToken:
		return f.jwt(ctx, method, txn)

	case model.MethodTypeSAPassword:
		return f.serviceAccountPass(ctx, method, txn)
	}

	f.logger.Warn(fmt.Sprintf("unsupported auth method %s", method.MethodType))
	return nil, nil, fmt.Errorf("unsupported auth method")
}

func (f *AuthenticatorFactory) getAuthSource(method *model.AuthMethod, txn *io.MemoryStoreTxn) (*model.AuthSource, error) {
	repo := repo.NewAuthSourceRepo(txn)
	authSource, err := repo.Get(method.Source)
	if err != nil {
		return nil, err
	}
	if authSource == nil {
		return nil, fmt.Errorf("not found auth method")
	}

	return authSource, nil
}

func (f *AuthenticatorFactory) jwt(ctx context.Context, method *model.AuthMethod, txn *io.MemoryStoreTxn) (authn2.Authenticator, *model.AuthSource, error) {
	authSource, err := f.getAuthSource(method, txn)
	if err != nil {
		return nil, nil, err
	}

	jwtValidator, err := f.jwtValidator(ctx, method, authSource)
	if err != nil {
		return nil, nil, err
	}

	return &jwt2.Authenticator{
		AuthMethod:   method,
		Logger:       f.logger.Named("JWTAutheNticator"),
		AuthSource:   authSource,
		JwtValidator: jwtValidator,
	}, authSource, nil
}

func (f *AuthenticatorFactory) multipass(ctx context.Context, method *model.AuthMethod, txn *io.MemoryStoreTxn) (authn2.Authenticator, *model.AuthSource, error) {
	f.logger.Debug("Check jwt is enabled")

	enabled, err := f.jwtController.IsEnabled(txn)
	if err != nil {
		return nil, nil, err
	}
	if !enabled {
		f.logger.Warn("jwt is not enabled. Do not use multipass login")
		return nil, nil, fmt.Errorf("jwt is not enabled. not use multipass login")
	}

	f.logger.Debug("Jwt is enabled. Get jwt public keys")

	keys, err := f.jwtController.JWKS(txn)
	if err != nil {
		return nil, nil, err
	}

	f.logger.Debug(fmt.Sprintf("Got jwt keys. %s Get jwt config", keys))

	jwtConf, err := f.jwtController.GetConfig(txn)
	if err != nil {
		return nil, nil, err
	}

	f.logger.Debug("Got jwt config")

	authSource := model.GetMultipassSourceForLogin(jwtConf, keys)
	jwtValidator, err := f.jwtValidator(ctx, method, authSource)
	if err != nil {
		return nil, nil, err
	}

	loggerForAuth := f.logger.Named("MultipassAutheNticator")

	authenticator := &multipass2.Authenticator{
		AuthSource:   authSource,
		AuthMethod:   method,
		JwtValidator: jwtValidator,
		Logger:       loggerForAuth,
		MultipassService: &usecase.Multipass{
			JwtController:    f.jwtController,
			MultipassRepo:    iam_repo.NewMultipassRepository(txn),
			GenMultipassRepo: repo.NewMultipassGenerationNumberRepository(txn),
			Logger:           loggerForAuth,
		},
	}

	return authenticator, authSource, nil
}

// jwtValidator returns a new JWT validator based on the provided config.
func (f *AuthenticatorFactory) jwtValidator(ctx context.Context, method *model.AuthMethod, config *model.AuthSource) (jwt2.JwtValidator, error) {
	f.l.Lock()
	defer f.l.Unlock()

	if v, ok := f.validators[method.Name]; ok {
		return v, nil
	}

	if method.MethodType == model.MethodTypeAccessToken {
		validator, err := accesstoken.NewAccessTokenValidator(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("%w: access token validator configuration error: %s", consts.ErrNotConfigured, err.Error())
		}
		f.validators[method.Name] = validator
		return validator, nil
	}

	var err error
	var keySet hcjwt.KeySet

	// Configure the key set for the pure jwt validator
	switch config.AuthType() {
	case model.AuthSourceJWKS:
		keySet, err = hcjwt.NewJSONWebKeySet(ctx, config.JWKSURL, config.JWKSCAPEM)
	case model.AuthSourceStaticKeys:
		keySet, err = hcjwt.NewStaticKeySet(config.ParsedJWTPubKeys)
	case model.AuthSourceOIDCDiscovery:
		keySet, err = hcjwt.NewOIDCDiscoveryKeySet(ctx, config.OIDCDiscoveryURL, config.OIDCDiscoveryCAPEM)
	default:
		return nil, fmt.Errorf("%w: unsupported config type", consts.ErrNotConfigured)
	}

	if err != nil {
		return nil, fmt.Errorf("%w: keyset configuration error: %s", consts.ErrNotConfigured, err.Error())
	}

	validator, err := hcjwt.NewValidator(keySet)
	if err != nil {
		return nil, fmt.Errorf("%w: JWT validator configuration error: %s", consts.ErrNotConfigured, err.Error())
	}

	// not cache multipass validator
	// TODO flush cache when update JWKS
	// TODO flush cache when update authsource (ie CAPEM)
	if config.UUID != model.MultipassSourceUUID {
		f.validators[method.Name] = validator
	}

	return validator, nil
}

func (f *AuthenticatorFactory) serviceAccountPass(ctx context.Context, method *model.AuthMethod, txn *io.MemoryStoreTxn) (authn2.Authenticator, *model.AuthSource, error) {
	authSource := model.GetServiceAccountPassSource()
	return &serviceaccountpass.Authenticator{
		ServiceAccountPasswordRepo: iam_repo.NewServiceAccountPasswordRepository(txn),
		AuthMethod:                 method,
		Logger:                     f.logger.Named("SAPassAutheNticator"),
	}, authSource, nil
}
