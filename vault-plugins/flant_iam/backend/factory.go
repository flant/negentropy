package backend

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
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

func newBackend(conf *logical.BackendConfig) (logical.Backend, error) {
	b := &framework.Backend{
		Help:        strings.TrimSpace(commonHelp),
		BackendType: logical.TypeLogical,
	}

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), conf.StorageView, "root_source")
	if err != nil {
		return nil, err
	}

	schema, err := model.GetSchema()
	if err != nil {
		return nil, err
	}

	storage, err := sharedio.NewMemoryStore(schema, mb)
	if err != nil {
		return nil, err
	}
	storage.SetLogger(conf.Logger)

	storage.AddKafkaSource(kafka_source.NewMainKafkaSource(mb, "root_source"))

	err = storage.Restore()
	if err != nil {
		return nil, err
	}

	// destinations
	storage.AddKafkaDestination(kafka_destination.NewMainKafkaDestination(mb, "root_source"))
	replicaIter, err := storage.Txn(false).Get(model.ReplicaType, model.ID)
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
			log.Println("unknown replica type: ", replica.Name, replica.TopicType)
		}
	}

	b.Paths = framework.PathAppend(
		tenantPaths(b, storage),
		userPaths(b, storage),
		projectPaths(b, storage),

		replicasPaths(b, storage),
		mb.KafkaPaths(),
	)

	return b, nil
}

const commonHelp = `
IAM API here
`
