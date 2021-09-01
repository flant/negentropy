package backend

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/cap/oidc"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/patrickmn/go-cache"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	extension_server_access2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	entity_api "github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/root"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/self"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	factory2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn/factory"
	"github.com/flant/negentropy/vault-plugins/shared/client"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	njwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/openapi"
)

const loggerModule = "Auth"

// Factory is used by framework
func Factory(ctx context.Context, c *logical.BackendConfig, jwksIDGetter func() (string, error)) (logical.Backend, error) {
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

	jwtController         *njwt.Controller
	accessVaultController *client.VaultClientController

	storage *sharedio.MemoryStore

	accessorGetter *vault.MountAccessorGetter

	jwksIdGetter func() (string, error)
}

func backend(conf *logical.BackendConfig, jwksIDGetter func() (string, error)) (*flantIamAuthBackend, error) {
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

	schema, err := repo.GetSchema()
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
	storage.AddKafkaSource(kafka_source.NewMultipassGenerationSource(mb, iamAuthLogger.Named("KafkaSourceMultipassGen")))

	err = storage.Restore()
	if err != nil {
		return nil, err
	}

	storage.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))
	storage.AddKafkaDestination(jwtkafka.NewJWKSKafkaDestination(mb, iamAuthLogger.Named("KafkaDestinationJWKS")))
	storage.AddKafkaDestination(kafka_destination.NewMultipassGenerationKafkaDestination(mb, iamAuthLogger.Named("KafkaDestinationMultipassGen")))

	if jwksIDGetter == nil {
		jwksIDGetter = mb.GetEncryptionPublicKeyStrict
	}

	b.storage = storage
	b.jwtController = njwt.NewJwtController(
		storage,
		jwksIDGetter,
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

	b.authnFactoty = factory2.NewAuthenticatorFactory(b.jwtController, iamAuthLogger.Named("Login"))

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
				pathMultipassOwner(b),
				pathJwtType(b),
				pathJwtTypeList(b),
				pathIssueJwtType(b),
				pathIssueMultipassJwt(b),

				// Uncomment to mount simple UI handler for local development
				// pathUI(b),
			},

			b.jwtController.ApiPaths(),

			[]*framework.Path{
				client.PathConfigure(b.accessVaultController),
			},
			pathOIDC(b),
			kafkaPaths(b, storage, iamAuthLogger.Named("KafkaBackend")),

			// server_access_extension
			extension_server_access2.ServerAccessPaths(b, storage, b.jwtController),
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

// revealEntityIDOwner returns type and info about token owner
// it can be iam.User, or iam.ServiceAccount
func (b *flantIamAuthBackend) revealEntityIDOwner(ctx context.Context,
	req *logical.Request) (string, interface{}, error) {
	logger := b.NamedLogger("revealEntityIDOwner")
	logger.Debug(fmt.Sprintf("EntityID=%s", req.EntityID))
	vaultClient, err := b.accessVaultController.APIClient()
	if err != nil {
		return "", nil, fmt.Errorf("internal error accessing vault client: %w", err)
	}

	entityApi := entity_api.NewIdentityAPI(vaultClient, logger.Named("LoginIdentityApi")).EntityApi()
	ent, err := entityApi.GetByID(req.EntityID)
	if err != nil {
		return "", nil, fmt.Errorf("finding vault entity by id: %w", err)
	}

	name, ok := ent["name"]
	if !ok {
		return "", nil, fmt.Errorf("field 'name' in vault entity is ommited")
	}

	nameStr, ok := name.(string)
	if !ok {
		return "", nil, fmt.Errorf("field 'name' should be string")
	}
	logger.Debug(fmt.Sprintf("catch name of vault entity: %s", nameStr))

	txn := b.storage.Txn(false)
	defer txn.Abort()

	iamEntity, err := repo.NewEntityRepo(txn).GetByName(nameStr)
	if err != nil {
		return "", nil, fmt.Errorf("finding iam_entity by name:%w", err)
	}
	logger.Debug(fmt.Sprintf("catch multipass owner UUID: %s, try to find user", iamEntity.UserId))

	user, err := iam_repo.NewUserRepository(txn).GetByID(iamEntity.UserId)
	if err != nil && !errors.Is(err, iam.ErrNotFound) {
		return "", nil, fmt.Errorf("finding user by id:%w", err)
	}
	if err == nil {
		logger.Debug(fmt.Sprintf("found user UUID: %s", user.UUID))
		return iam.UserType, user, nil
	} else {
		logger.Debug("Not found user, try to find service_account")
		sa, err := iam_repo.NewServiceAccountRepository(txn).GetByID(iamEntity.UUID)
		if err != nil && !errors.Is(err, iam.ErrNotFound) {
			return "", nil, fmt.Errorf("finding service_account by id:%w", err)
		}
		if errors.Is(err, iam.ErrNotFound) {
			logger.Debug("Not found neither user nor service_account")
			return "", nil, err
		}
		logger.Debug(fmt.Sprintf("found service_account UUID: %s", sa.UUID))
		return iam.ServiceAccountType, sa, nil
	}
}

// availableTenantsAndProjectsByEntityIDOwner returns sets of tenants and projects available for EntityIDOwner
func (b *flantIamAuthBackend) availableTenantsAndProjectsByEntityIDOwner(ctx context.Context,
	req *logical.Request) (map[iam.TenantUUID]struct{}, map[iam.ProjectUUID]struct{}, error) {
	subjectType, subject, err := b.revealEntityIDOwner(ctx, req)
	if errors.Is(err, iam.ErrNotFound) {
		return map[iam.TenantUUID]struct{}{}, map[iam.ProjectUUID]struct{}{}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	switch subjectType {
	case iam.UserType:
		{
			user, ok := subject.(*iam.User)
			if !ok {
				return nil, nil, fmt.Errorf("can't cast, need *model.User, got: %T", subject)
			}
			txn := b.storage.Txn(false)
			defer txn.Abort()
			groups, err := iam_repo.NewGroupRepository(txn).FindAllParentGroupsForUserUUID(user.UUID)
			gs := make([]iam.GroupUUID, 0, len(groups))
			for uuid := range groups {
				gs = append(gs, uuid)
			}
			if err != nil {
				return nil, nil, fmt.Errorf("collecting tenants, get groups: %w", err)
			}
			//  TODO Here easy to have two or three paralleled goroutines
			tenants, err := iam_repo.NewIdentitySharingRepository(txn).ListDestinationTenantsByGroupUUIDs(gs...)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting tenants, get target tenants: %w", err)
			}
			tenants[user.TenantUUID] = struct{}{}

			rbRepository := iam_repo.NewRoleBindingRepository(txn)
			userRBs, err := rbRepository.FindDirectRoleBindingsForUser(user.UUID)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects, get FindDirectRoleBindingsForUser: %w", err)
			}
			groupsRBs, err := rbRepository.FindDirectRoleBindingsForGroups(gs...)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects, get FindDirectRoleBindingsForGroups: %w", err)
			}
			projects, err := collectProjectUUIDsFromRoleBindings(userRBs, groupsRBs, txn)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects: %w", err)
			}
			return tenants, projects, nil
		}

	case iam.ServiceAccountType:
		{
			sa, ok := subject.(*iam.ServiceAccount)
			if !ok {
				return nil, nil, fmt.Errorf("can't cast, need *model.ServiceAccount, got: %T", subject)
			}
			txn := b.storage.Txn(false)
			defer txn.Abort()
			groups, err := iam_repo.NewGroupRepository(txn).FindAllParentGroupsForServiceAccountUUID(sa.UUID)
			gs := make([]iam.GroupUUID, 0, len(groups))
			for uuid := range groups {
				gs = append(gs, uuid)
			}
			if err != nil {
				return nil, nil, fmt.Errorf("collecting tenants, get groups: %w", err)
			}
			//  TODO Here easy to have two or three paralleled goroutines
			tenants, err := iam_repo.NewIdentitySharingRepository(txn).ListDestinationTenantsByGroupUUIDs(gs...)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting tenants, get target tenants: %w", err)
			}
			tenants[sa.TenantUUID] = struct{}{}
			rbRepository := iam_repo.NewRoleBindingRepository(txn)

			userRBs, err := rbRepository.FindDirectRoleBindingsForServiceAccount(sa.UUID)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects, get FindDirectRoleBindingsForServiceAccount: %w", err)
			}
			groupsRBs, err := rbRepository.FindDirectRoleBindingsForGroups(gs...)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects, get FindDirectRoleBindingsForGroups: %w", err)
			}
			projects, err := collectProjectUUIDsFromRoleBindings(userRBs, groupsRBs, txn)
			if err != nil {
				return nil, nil, fmt.Errorf("collecting projects: %w", err)
			}
			return tenants, projects, nil
		}
	}
	return nil, nil, fmt.Errorf("unexpected subjectType: `%s`", subjectType)
}

func collectProjectUUIDsFromRoleBindings(rbs1 map[iam.RoleBindingUUID]*iam.RoleBinding,
	rbs2 map[iam.RoleBindingUUID]*iam.RoleBinding, txn *sharedio.MemoryStoreTxn) (map[iam.ProjectUUID]struct{}, error) {
	result := map[iam.ProjectUUID]struct{}{}
	projectRepo := iam_repo.NewProjectRepository(txn)
	fullTenants := map[iam.TenantUUID]struct{}{}
	for _, rb := range rbs1 {
		err := processRoleBinding(rb, &fullTenants, projectRepo, &result)
		if err != nil {
			return nil, err
		}
	}
	for _, rb := range rbs2 {
		err := processRoleBinding(rb, &fullTenants, projectRepo, &result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// processRoleBinding check rb and write to given pointers
func processRoleBinding(rb *iam.RoleBinding, fullTenants *map[iam.TenantUUID]struct{},
	projectRepo *iam_repo.ProjectRepository, result *map[iam.ProjectUUID]struct{}) error {
	if rb.AnyProject {
		if _, processedTenant := (*fullTenants)[rb.TenantUUID]; !processedTenant {
			(*fullTenants)[rb.TenantUUID] = struct{}{}
			allTenantProject, err := projectRepo.ListIDs(rb.TenantUUID, false)
			if err != nil {
				return err
			}
			for _, p := range allTenantProject {
				(*result)[p] = struct{}{}
			}
		}
	} else {
		for _, pUUID := range rb.Projects {
			(*result)[pUUID] = struct{}{}
		}
	}
	return nil
}
