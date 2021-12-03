package backend

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	ext_ff_io "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/io"
	ext_ff_paths "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access"
	ext_sa_io "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/io"
	ext_sa_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	sharedjwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	baseLogger := conf.Logger.ResetNamed("IAM")

	logger := baseLogger.Named("Factory")
	logger.Debug("started")
	defer logger.Debug("exit")
	if os.Getenv("DEBUG") == "true" {
		logger.Debug("DEBUG mode is ON, messages will not be encrypted")
		sharedkafka.DoNotEncrypt = true
	}

	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	conf.Logger = baseLogger

	b, err := newBackend(ctx, conf)
	if err != nil {
		return nil, err
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	logger.Debug("normal finish")
	return b, nil
}

type ExtensionInitializationFunc func(ctx context.Context, initRequest *logical.InitializationRequest, storage *sharedio.MemoryStore) error

func initializer(storage *sharedio.MemoryStore) func(ctx context.Context, initRequest *logical.InitializationRequest) error {
	return func(ctx context.Context, initRequest *logical.InitializationRequest) error {
		initFuncs := []ExtensionInitializationFunc{
			ext_server_access.InitializeExtensionServerAccess,
		}

		for _, f := range initFuncs {
			err := f(ctx, initRequest, storage)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func newBackend(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := &framework.Backend{
		Help:        strings.TrimSpace(commonHelp),
		BackendType: logical.TypeLogical,
	}

	logger := conf.Logger.Named("newBackend")
	logger.Debug("started")
	defer logger.Debug("exit")

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), conf.StorageView, conf.Logger)
	if err != nil {
		return nil, err
	}

	iamSchema, err := iam_repo.GetSchema()
	if err != nil {
		return nil, err
	}

	schema, err := memdb.MergeDBSchemasAndValidate(iamSchema, ext_sa_repo.ServerSchema(), ext_ff_paths.FlantFlowDBSchema())
	if err != nil {
		return nil, err
	}

	storage, err := sharedio.NewMemoryStore(schema, mb, conf.Logger)
	if err != nil {
		return nil, err
	}

	if backentutils.IsLoading(conf) {
		logger.Info("final run Factory, apply kafka operations on MemoryStore")

		restoreHandlers := []kafka_source.RestoreFunc{
			jwtkafka.SelfRestoreMessage,
			ext_sa_io.HandleServerAccessObjects,
			ext_ff_io.HandleFlantFlowObjects,
		}

		storage.AddKafkaSource(kafka_source.NewSelfKafkaSource(mb, restoreHandlers, conf.Logger))

		storage.AddKafkaSource(jwtkafka.NewJWKSKafkaSource(mb, conf.Logger))

		err = storage.Restore()
		if err != nil {
			return nil, err
		}

		// destinations
		storage.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))

		storage.AddKafkaDestination(jwtkafka.NewJWKSKafkaDestination(mb, conf.Logger))

		storage.RunKafkaSourceMainLoops()

	} else {
		logger.Info("first run Factory, skipping kafka operations on MemoryStore")
	}

	replicaIter, err := storage.Txn(false).Get(model.ReplicaType, iam_repo.PK)
	if err != nil {
		return nil, err
	}

	for {
		raw := replicaIter.Next()
		if raw == nil {
			break
		}

		replica := raw.(*model.Replica)
		switch replica.TopicType {
		case kafka_destination.VaultTopicType:
			storage.AddKafkaDestination(kafka_destination.NewVaultKafkaDestination(mb, *replica))
		case kafka_destination.MetadataTopicType:
			storage.AddKafkaDestination(kafka_destination.NewMetadataKafkaDestination(mb, *replica))
		default:
			log.L().Warn("unknown replica type: ", replica.Name, replica.TopicType)
		}
	}

	b.InitializeFunc = initializer(storage)

	tokenController := sharedjwt.NewJwtController(
		storage,
		mb.GetEncryptionPublicKeyStrict,
		conf.Logger,
		time.Now,
	)

	b.PeriodicFunc = func(ctx context.Context, request *logical.Request) error {
		periodicLogger := conf.Logger.Named("Periodic")
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

		run("jwtController", func() error {
			tx := storage.Txn(true)
			defer tx.Abort()

			err := tokenController.OnPeriodical(tx)
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

	flantFlowPaths, err := ext_ff_paths.FlantFlowPaths(ctx, conf, storage)
	if err != nil {
		return nil, err
	}

	b.Paths = framework.PathAppend(
		tenantPaths(b, storage),

		userPaths(b, tokenController, storage),
		serviceAccountPaths(b, tokenController, storage),

		groupPaths(b, storage),
		projectPaths(b, storage),
		featureFlagPaths(b, storage),
		roleBindingPaths(b, storage),
		roleBindingApprovalPaths(b, storage),
		rolePaths(b, storage),

		replicasPaths(b, storage),
		kafkaPaths(b, storage, conf.Logger),
		identitySharingPaths(b, storage),

		ext_server_access.ServerPaths(b, storage, tokenController),
		ext_server_access.ServerConfigurePaths(b, storage),

		tokenController.ApiPaths(),

		flantFlowPaths,
	)

	b.Clean = func(context.Context) {
		l := b.Logger().Named("cleanup")
		l.Info("started")

		storage.Close()

		l.Info("finished")
	}

	logger.Debug("normal finish")
	return b, nil
}

const commonHelp = `
IAM API here
`
