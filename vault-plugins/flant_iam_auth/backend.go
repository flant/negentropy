package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"sync"
	"time"

	"github.com/hashicorp/cap/jwt"
	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/patrickmn/go-cache"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extension_server_access"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/root"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/self"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/client"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	njwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/openapi"
)

const loggerModule = "IamAuth"

// Factory is used by framework
func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b, err := backend(c)
	if err != nil {
		return nil, err
	}
	if err := b.SetupBackend(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

type flantIamAuthBackend struct {
	*framework.Backend

	l                  sync.RWMutex
	provider           *oidc.Provider
	validators         map[string]*jwt.Validator
	oidcRequests       *cache.Cache
	jwtTypesValidators map[string]openapi.Validator

	providerCtx       context.Context
	providerCtxCancel context.CancelFunc

	jwtController         *njwt.Controller
	accessVaultController *client.VaultClientController

	storage *sharedio.MemoryStore

	accessorGetter *vault.MountAccessorGetter
}

func backend(conf *logical.BackendConfig) (*flantIamAuthBackend, error) {
	b := new(flantIamAuthBackend)
	b.jwtTypesValidators = map[string]openapi.Validator{}
	b.providerCtx, b.providerCtxCancel = context.WithCancel(context.Background())
	b.oidcRequests = cache.New(oidcRequestTimeout, oidcRequestCleanupInterval)

	iamAuthLogger := conf.Logger.Named(loggerModule)

	b.accessVaultController = client.NewVaultClientController(func() hclog.Logger {
		return iamAuthLogger.Named("ApiClient")
	})

	mb, err := kafka.NewMessageBroker(context.TODO(), conf.StorageView)
	if err != nil {
		return nil, err
	}

	schema, err := model.GetSchema()
	if err != nil {
		return nil, err
	}

	clientGetter := func() (*api.Client, error) {
		return b.accessVaultController.APIClient()
	}

	b.accessorGetter = vault.NewMountAccessorGetter(clientGetter, "flant_iam_auth/")
	entityApi := vault.NewVaultEntityDownstreamApi(clientGetter, b.accessorGetter, iamAuthLogger.Named("VaultIdentityClient"))

	storage, err := sharedio.NewMemoryStore(schema, mb)
	if err != nil {
		return nil, err
	}
	storage.SetLogger(iamAuthLogger.Named("MemStorage"))

	selfSourceHandlerLogger := iamAuthLogger.Named("SelfSourceHandler")
	selfSourceHandler := func(store *sharedio.MemoryStore, tx *sharedio.MemoryStoreTxn) self.ModelHandler {
		return self.NewObjectHandler(store, tx, entityApi, selfSourceHandlerLogger)
	}

	rootSourceHandlerLogger := iamAuthLogger.Named("RootSourceHandler")
	rootSourceHandler := func(tx *sharedio.MemoryStoreTxn) root.ModelHandler {
		return root.NewObjectHandler(tx, rootSourceHandlerLogger)
	}

	storage.AddKafkaSource(kafka_source.NewSelfKafkaSource(mb, selfSourceHandler, iamAuthLogger.Named("KafkaSourceSelf")))
	storage.AddKafkaSource(kafka_source.NewRootKafkaSource(mb, rootSourceHandler, iamAuthLogger.Named("KafkaSourceRoot")))
	storage.AddKafkaSource(jwtkafka.NewJWKSKafkaSource(mb, iamAuthLogger.Named("KafkaSourceJWKS")))

	err = storage.Restore()
	if err != nil {
		return nil, err
	}

	storage.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))
	storage.AddKafkaDestination(jwtkafka.NewJWKSKafkaDestination(mb))

	b.storage = storage
	b.jwtController = njwt.NewJwtController(
		storage,
		mb.GetEncryptionPublicKeyStrict,
		iamAuthLogger.Named("Jwt"),
		time.Now,
	)

	periodicLogger := iamAuthLogger.Named("PeriodicFunc")
	periodicFunction := func(ctx context.Context, request *logical.Request) error {
		periodicLogger.Debug("Run periodic function")
		defer periodicLogger.Debug("Periodic function finished")

		var allErrors *multierror.Error
		run := func(controllerName string, action func() error) {
			periodicLogger.Debug(fmt.Sprintf("Run %s OnPeriodical", controllerName))

			err := action()

			if err != nil {
				allErrors = multierror.Append(allErrors, err)
				periodicLogger.Error(fmt.Sprintf("Error while %s periodical function: %s", controllerName, err), "err", err)
				return
			}

			periodicLogger.Debug(fmt.Sprintf("%s OnPeriodical was successful run", controllerName))
		}

		run("accessVaultController", func() error {
			return b.accessVaultController.OnPeriodical(ctx, request)
		})

		run("jwtController", func() error {
			tx := b.storage.Txn(true)
			defer tx.Abort()

			err := b.jwtController.OnPeriodical(tx)
			if err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				periodicLogger.Error(fmt.Sprintf("Can not commit memdb transaction: %s", err), "err", err)
				return err
			}

			return nil
		})

		return allErrors
	}

	b.Backend = &framework.Backend{
		AuthRenew:   b.pathLoginRenew,
		BackendType: logical.TypeCredential,
		Invalidate:  b.invalidate,
		PeriodicFunc: periodicFunction,
		Help: backendHelp,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{
				"login",
				"oidc/auth_url",
				"oidc/callback",
				"jwks",
				// Uncomment to mount simple UI handler for local development
				// "ui",
			},
			SealWrapStorage: []string{
				"config",
			},
		},
		Paths: framework.PathAppend(
			[]*framework.Path{
				pathAuthMethodList(b),
				pathAuthMethod(b),
				pathAuthSource(b),
				pathAuthSourceList(b),
				pathLogin(b),
				pathJwtType(b),
				pathJwtTypeList(b),
				pathIssueJwtType(b),

				// Uncomment to mount simple UI handler for local development
				// pathUI(b),
			},

			b.jwtController.ApiPaths(),

			[]*framework.Path{
				client.PathConfigure(b.accessVaultController),
			},
			pathOIDC(b),
			kafkaPaths(b, storage),

			// server_access_extension
			extension_server_access.ServerAccessPaths(b, storage, b.jwtController),
			pathTenant(b),
		),
		Clean: b.cleanup,
	}

	return b, nil
}

func (b *flantIamAuthBackend) NamedLogger(name string) hclog.Logger {
	return b.Logger().Named(loggerModule).Named(name)
}

func (b *flantIamAuthBackend) SetupBackend(ctx context.Context, config *logical.BackendConfig) error {
	err := b.Setup(ctx, config)
	if err != nil {
		return err
	}

	err = b.accessVaultController.Init(config.StorageView)
	if err != nil && !errors.Is(err, client.ErrNotSetConf) {
		return err
	}

	return nil
}

func (b *flantIamAuthBackend) cleanup(_ context.Context) {
	b.l.Lock()
	if b.providerCtxCancel != nil {
		b.providerCtxCancel()
	}
	if b.provider != nil {
		b.provider.Done()
	}
	b.l.Unlock()
}

func (b *flantIamAuthBackend) invalidate(ctx context.Context, key string) {
	switch key {
	case "config":
		b.reset()
	default:
		return
	}
}

func (b *flantIamAuthBackend) reset() {
	b.l.Lock()
	if b.provider != nil {
		b.provider.Done()
	}
	b.provider = nil
	b.validators = make(map[string]*jwt.Validator)
	b.l.Unlock()
}

func (b *flantIamAuthBackend) getProvider(config *model.AuthSource) (*oidc.Provider, error) {
	b.l.Lock()
	defer b.l.Unlock()

	if b.provider != nil {
		return b.provider, nil
	}

	provider, err := b.createProvider(config)
	if err != nil {
		return nil, err
	}

	b.provider = provider
	return provider, nil
}

func (b *flantIamAuthBackend) jwtTypeValidator(jwtType *model.JWTIssueType) (openapi.Validator, error) {
	specStr := jwtType.OptionsSchema
	if specStr == "" {
		return nil, nil
	}

	validator := func() openapi.Validator {
		b.l.RLock()
		defer b.l.RUnlock()

		if spec, ok := b.jwtTypesValidators[jwtType.Name]; ok {
			return spec
		}

		return nil
	}()

	if validator != nil {
		return validator, nil
	}

	var err error
	validator, err = openapi.SchemaValidator(jwtType.OptionsSchema)
	if err != nil {
		return nil, err
	}

	b.setJWTTypeValidator(jwtType, validator)

	return validator, nil
}

func (b *flantIamAuthBackend) setJWTTypeValidator(jwtType *model.JWTIssueType, validator openapi.Validator) {
	b.l.Lock()
	defer b.l.Unlock()

	b.jwtTypesValidators[jwtType.Name] = validator
}

// jwtValidator returns a new JWT validator based on the provided config.
func (b *flantIamAuthBackend) jwtValidator(methodName string, config *model.AuthSource) (*jwt.Validator, error) {
	b.l.Lock()
	defer b.l.Unlock()

	if v, ok := b.validators[methodName]; ok {
		return v, nil
	}

	var err error
	var keySet jwt.KeySet

	// Configure the key set for the validator
	switch config.AuthType() {
	case model.AuthSourceJWKS:
		keySet, err = jwt.NewJSONWebKeySet(b.providerCtx, config.JWKSURL, config.JWKSCAPEM)
	case model.AuthSourceStaticKeys:
		keySet, err = jwt.NewStaticKeySet(config.ParsedJWTPubKeys)
	case model.AuthSourceOIDCDiscovery:
		keySet, err = jwt.NewOIDCDiscoveryKeySet(b.providerCtx, config.OIDCDiscoveryURL, config.OIDCDiscoveryCAPEM)
	default:
		return nil, errors.New("unsupported config type")
	}

	if err != nil {
		return nil, fmt.Errorf("keyset configuration error: %w", err)
	}

	validator, err := jwt.NewValidator(keySet)
	if err != nil {
		return nil, fmt.Errorf("JWT validator configuration error: %w", err)
	}

	b.validators[methodName] = validator

	return validator, nil
}

const (
	backendHelp = `
The JWT backend plugin allows authentication using JWTs (including OIDC).
`
)
