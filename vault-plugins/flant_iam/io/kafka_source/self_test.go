package kafka_source

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func TestInitial(t *testing.T) {
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

	err = mb.CreateTopic(context.TODO(), "root_source", nil)
	require.NoError(t, err)

	ss := NewSelfKafkaSource(mb, []RestoreFunc{})
	err = ss.Restore(nil, nil)
	require.NoError(t, err)
}
