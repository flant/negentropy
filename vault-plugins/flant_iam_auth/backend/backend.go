package backend

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/patrickmn/go-cache"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/root"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/self"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
	factory2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/factory"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/client"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	njwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/openapi"
)

// Factory is used by framework
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	baseLogger := conf.Logger.ResetNamed("AUTH")
	logger := baseLogger.Named("Factory")
	logger.Debug("started")
	defer logger.Debug("exit")
	if os.Getenv("DEBUG") == "true" {
		kafka.DoNotEncrypt = true
		logger.Debug("DEBUG mode is ON, messages will not be encrypted")
	}
	conf.Logger = baseLogger
	b, err := backend(conf, nil)
	if err != nil {
		return nil, err
	}
	if err := b.SetupBackend(ctx, conf); err != nil {
		return nil, err
	}
	logger.Debug("normal finish")
	return b, nil
}

func FactoryWithJwksIDGetter(ctx context.Context, c *logical.BackendConfig,
	jwksIDGetter func() (string, error)) (logical.Backend, error) {
	b, err := backend(c, jwksIDGetter)
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

	l            sync.RWMutex
	provider     *oidc.Provider
	oidcRequests *cache.Cache
	authnFactoty *factory2.AuthenticatorFactory

	jwtTypesValidators map[string]openapi.Validator

	providerCtx       context.Context
	providerCtxCancel context.CancelFunc

	jwtController       *njwt.Controller
	accessVaultProvider client.VaultClientController

	storage *sharedio.MemoryStore

	accessorGetter *vault.MountAccessorGetter

	jwksIdGetter func() (string, error)

	serverAccessBackend extension_server_access.ServerAccessBackend

	entityIDResolver authn.EntityIDResolver

	logger hclog.Logger
}

func backend(conf *logical.BackendConfig, jwksIDGetter func() (string, error)) (*flantIamAuthBackend, error) {
	b := new(flantIamAuthBackend)
	b.logger = conf.Logger
	logger := conf.Logger.Named("backend")
	logger.Debug("started")
	defer logger.Debug("exit")
	b.jwtTypesValidators = map[string]openapi.Validator{}
	b.providerCtx, b.providerCtxCancel = context.WithCancel(context.Background())
	b.oidcRequests = cache.New(oidcRequestTimeout, oidcRequestCleanupInterval)
	b.accessVaultProvider = client.NewVaultClientController(conf.Logger)
	mb, err := kafka.NewMessageBroker(context.TODO(), conf.StorageView, conf.Logger)
	if err != nil {
		return nil, err
	}
	schema, err := repo.GetSchema()
	if err != nil {
		return nil, err
	}

	b.accessorGetter = vault.NewMountAccessorGetter(b.accessVaultProvider, "flant/")
	entityApi := vault.NewVaultEntityDownstreamApi(b.accessVaultProvider, b.accessorGetter, conf.Logger)

	storage, err := sharedio.NewMemoryStore(schema, mb, conf.Logger)
	if err != nil {
		return nil, err
	}

	if backentutils.IsLoading(conf) {
		logger.Info("final run Factory, apply kafka operations on MemoryStore")

		storage.AddKafkaSource(kafka_source.NewSelfKafkaSource(mb, self.NewObjectHandler(entityApi, conf.Logger), conf.Logger))
		storage.AddKafkaSource(kafka_source.NewRootKafkaSource(mb, root.NewObjectHandler(conf.Logger), conf.Logger))
		storage.AddKafkaSource(jwtkafka.NewJWKSKafkaSource(mb, conf.Logger))
		storage.AddKafkaSource(kafka_source.NewMultipassGenerationSource(mb, conf.Logger))

		err = storage.Restore()
		if err != nil {
			return nil, err
		}

		storage.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))
		storage.AddKafkaDestination(jwtkafka.NewJWKSKafkaDestination(mb, conf.Logger))
		storage.AddKafkaDestination(kafka_destination.NewMultipassGenerationKafkaDestination(mb, conf.Logger))

		storage.RunKafkaSourceMainLoops()

	} else {
		logger.Info("first run Factory, skipping kafka operations on MemoryStore")
	}

	b.storage = storage

	if jwksIDGetter == nil {
		jwksIDGetter = mb.GetEncryptionPublicKeyStrict
	}

	b.jwtController = njwt.NewJwtController(
		storage,
		jwksIDGetter,
		conf.Logger,
		time.Now,
	)

	periodicFunction := func(ctx context.Context, request *logical.Request) error {
		periodicLogger := conf.Logger.Named("PeriodicFunc")
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

		run("accessVaultProvider", func() error {
			return b.accessVaultProvider.OnPeriodical(ctx, request)
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

	b.authnFactoty = factory2.NewAuthenticatorFactory(b.jwtController, conf.Logger)

	b.serverAccessBackend = extension_server_access.NewServerAccessBackend(b, storage)

	b.Backend = &framework.Backend{
		AuthRenew:    b.pathLoginRenew,
		BackendType:  logical.TypeCredential,
		Invalidate:   b.invalidate,
		PeriodicFunc: periodicFunction,
		Help:         backendHelp,
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
				pathCheckPermissions(b),
				pathVSTOwner(b),
				pathJwtType(b),
				pathJwtTypeList(b),
				pathIssueJwtType(b),
				pathIssueMultipassJwt(b),

				// Uncomment to mount simple UI handler for local development
				// pathUI(b),
			},

			policiesPaths(b, storage),

			b.jwtController.ApiPaths(),

			[]*framework.Path{
				client.PathConfigure(b.accessVaultProvider),
			},
			pathOIDC(b),
			kafkaPaths(b, storage, conf.Logger),

			// server_access_extension
			b.serverAccessBackend.Paths(),
			pathTenant(b),
		),
		Clean: b.cleanup,
	}
	logger.Debug("normal finish")

	return b, nil
}

func (b *flantIamAuthBackend) NamedLogger(name string) hclog.Logger {
	return b.logger.Named(name)
}

func (b *flantIamAuthBackend) SetupBackend(ctx context.Context, config *logical.BackendConfig) error {
	logger := config.Logger.Named("SetupBackend")
	logger.Debug("started")
	defer logger.Debug("exit")

	err := b.Setup(ctx, config)
	if err != nil {
		return err
	}

	b.entityIDResolver, err = authn.NewEntityIDResolver(b.Backend.Logger(), b.accessVaultProvider)
	if err != nil {
		return err
	}

	b.serverAccessBackend.SetEntityIDResolver(b.entityIDResolver)
	authz.RunVaultPoliciesGarbageCollector(b.accessVaultProvider, config.Logger.Named("VaultPoliciesGarbageCollector"))

	logger.Debug("normal finish")
	return nil
}

func (b *flantIamAuthBackend) cleanup(_ context.Context) {
	l := b.Logger().Named("cleanup")
	l.Info("started")
	b.l.Lock()
	if b.providerCtxCancel != nil {
		b.providerCtxCancel()
	}
	if b.provider != nil {
		b.provider.Done()
	}
	b.storage.Close()
	b.l.Unlock()
	l.Info("finished")
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
	b.authnFactoty.Reset()
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

const (
	backendHelp = `
The JWT backend plugin allows authentication using JWTs (including OIDC).
`
)
