package backend

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func TestCommit(t *testing.T) {
	storage := &logical.InmemStorage{}
	key := "kafka.config"
	pl := "kafka.plugin.config"

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	config := kafka.BrokerConfig{
		Endpoints:             []string{"localhost:9093"},
		ConnectionPrivateKey:  nil,
		ConnectionCertificate: nil,
		EncryptionPrivateKey:  pk,
		EncryptionPublicKey:   &pk.PublicKey,
	}

	plugin := kafka.PluginConfig{
		SelfTopicName: "root_source",
	}
	d1, _ := json.Marshal(config)
	d2, _ := json.Marshal(plugin)
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: key, Value: d1})
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: pl, Value: d2})

	mb, err := kafka.NewMessageBroker(context.TODO(), storage)
	require.NoError(t, err)

	err = mb.CreateTopic("root_source")
	require.NoError(t, err)

	sc, err := model.GetSchema()
	require.NoError(t, err)

	memstore, err := io.NewMemoryStore(sc, mb)
	require.NoError(t, err)
	memstore.AddKafkaDestination(kafka_destination.NewSelfKafkaDestination(mb))

	tx := memstore.Txn(true)
	tenant := &model.Tenant{
		UUID:       "7ba17ad1-85f2-42d7-ad27-993ff026ef24",
		Version:    "1",
		Identifier: "2",
	}
	err = tx.Insert(model.TenantType, tenant)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)
}
