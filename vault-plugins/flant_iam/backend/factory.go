package backend

import (
	"context"
	"fmt"
	"strings"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access"

	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	sharedjwt "github.com/flant/negentropy/vault-plugins/shared/jwt"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
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

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), conf.StorageView)
	if err != nil {
		return nil, err
	}

	schema, err := model.GetSchema()
	if err != nil {
		return nil, err
	}

	schema, err = model2.MergeSchema(schema)
	if err != nil {
		return nil, err
	}

	storage, err := sharedio.NewMemoryStore(schema, mb)
	if err != nil {
		return nil, err
	}
	storage.SetLogger(conf.Logger)

	storage.AddKafkaSource(kafka_source.NewSelfKafkaSource(mb))
	storage.AddKafkaSource(jwtkafka.NewJWKSKafkaSource(mb, conf.Logger.Named("KafkaSourceJWKS")))

	err = storage.Restore()
	if err != nil {
		return nil, err
	}

	// destinations
	storage.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))
	storage.AddKafkaDestination(jwtkafka.NewJWKSKafkaDestination(mb))
	replicaIter, err := storage.Txn(false).Get(model.ReplicaType, model.PK)
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

	tokenController := sharedjwt.NewTokenController()

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
		kafkaPaths(b, storage),
		identitySharingPaths(b, storage),
		[]*framework.Path{
			sharedjwt.PathEnable(tokenController),
			sharedjwt.PathDisable(tokenController),
			sharedjwt.PathConfigure(tokenController),
			sharedjwt.PathJWKS(tokenController),
			sharedjwt.PathRotateKey(tokenController),
		},

		extension_server_access.ServerPaths(b, storage),
		extension_server_access.ServerConfigurePaths(b, storage),
	)

	return b, nil
}

const commonHelp = `
IAM API here
`
