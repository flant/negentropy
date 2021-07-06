package backend

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/explugin/io/kafka_destination"
	"github.com/flant/negentropy/explugin/io/kafka_source"
	"github.com/flant/negentropy/explugin/model"
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

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), conf.StorageView)
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

	storage.AddKafkaSource(kafka_source.NewSelfKafkaSource(mb))

	err = storage.Restore()
	if err != nil {
		return nil, err
	}

	// destinations
	storage.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))
	if mb.PluginConfig.RootPublicKey != nil {
		storage.AddKafkaDestination(kafka_destination.NewIAMKafkaDestination(mb, mb.PluginConfig.RootPublicKey, mb.PluginConfig.SelfTopicName))
	}

	b.Paths = framework.PathAppend(
		kafkaPaths(b, storage),
		userPaths(b, storage),
	)

	return b, nil
}

const commonHelp = `
Test API here
`
