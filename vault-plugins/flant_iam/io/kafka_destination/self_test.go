package kafka_destination

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func TestSendItem(t *testing.T) {
	// t.Skip("manual or integration test. Requires kafka")
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

	ss := NewSelfKafkaDestination(mb)
	u := &model.User{
		UUID:           "asdasd",
		TenantUUID:     "asdasd",
		Version:        "2",
		Identifier:     "3",
		FullIdentifier: "4",
	}
	msg, err := ss.ProcessObject(nil, nil, u)
	require.NoError(t, err)
	fmt.Println(msg)

	err = mb.SendMessages(msg, nil)
	require.NoError(t, err)
}
