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

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access"
	ext_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	sharedjwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	logger := conf.Logger.Named("flant_iam.Factory")
	logger.Debug("started")
	defer logger.Debug("exit")
	if os.Getenv("DEBUG") == "true" {
		logger.Debug("DEBUG mode is ON, messages will not be encrypted")
		sharedkafka.DoNotEncrypt = true
	}

	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	b, err := newBackend(conf)
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
			extension_server_access.InitializeExtensionServerAccess,
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

func newBackend(conf *logical.BackendConfig) (logical.Backend, error) {
	b := &framework.Backend{
		Help:        strings.TrimSpace(commonHelp),
		BackendType: logical.TypeLogical,
	}

	localLogger := conf.Logger.Named("newBackend")
	localLogger.Debug("started")
	defer localLogger.Debug("exit")

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), conf.StorageView, conf.Logger)
	if err != nil {
		return nil, err
	}

	schema, err := iam_repo.GetSchema()
	if err != nil {
		return nil, err
	}

	schema, err = ext_repo.MergeSchema(schema)
	if err != nil {
		return nil, err
	}

	storage, err := sharedio.NewMemoryStore(schema, mb)
	if err != nil {
		return nil, err
	}
	storage.SetLogger(conf.Logger)

	restoreHandlers := []kafka_source.RestoreFunc{
		jwtkafka.SelfRestoreMessage,
	}

	logger := conf.Logger.Named("IAM")

	storage.AddKafkaSource(kafka_source.NewSelfKafkaSource(mb, restoreHandlers, conf.Logger.Named("KafkaSourceSelf")))

	storage.AddKafkaSource(jwtkafka.NewJWKSKafkaSource(mb, conf.Logger.Named("KafkaSourceJWKS")))

	err = storage.Restore()
	if err != nil {
		return nil, err
	}

	// destinations
	storage.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))

	storage.AddKafkaDestination(jwtkafka.NewJWKSKafkaDestination(mb, logger.Named("KafkaSourceJWKS")))

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
		logger.Named("JWT.Controller"),
		time.Now,
	)

	periodicLogger := logger.Named("Periodic")
	b.PeriodicFunc = func(ctx context.Context, request *logical.Request) error {
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
		kafkaPaths(b, storage, logger.Named("KafkaPath")),
		identitySharingPaths(b, storage),

		extension_server_access.ServerPaths(b, storage, tokenController),
		extension_server_access.ServerConfigurePaths(b, storage),

		tokenController.ApiPaths(),
	)
	localLogger.Debug("normal finish")

	b.Clean = func(context.Context) {
		l := b.Logger().Named("cleanup")
		l.Info("started")

		storage.Close()

		l.Info("finished")
	}

	return b, nil
}

const commonHelp = `
IAM API here
`
