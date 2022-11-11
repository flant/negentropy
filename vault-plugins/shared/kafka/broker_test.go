package kafka

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalingBrokerConfig(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		var config BrokerConfig

		data := []byte("{}")
		err := json.Unmarshal(data, &config)
		require.NoError(t, err)
	})

	t.Run("existent config", func(t *testing.T) {
		rpriv, err := rsa.GenerateKey(rand.Reader, 4096)
		require.NoError(t, err)

		storedConfig := BrokerConfig{
			Endpoints:            []string{"localhost:9093", "localhost:9094"},
			EncryptionPrivateKey: rpriv,
			EncryptionPublicKey:  &rpriv.PublicKey,
		}

		d2, err := json.Marshal(storedConfig)
		require.NoError(t, err)

		var newConfig BrokerConfig

		err = json.Unmarshal(d2, &newConfig)
		require.NoError(t, err)
		assert.Equal(t, storedConfig.Endpoints, newConfig.Endpoints)
		assert.Equal(t, storedConfig.EncryptionPublicKey, newConfig.EncryptionPublicKey)
		assert.Equal(t, storedConfig.EncryptionPrivateKey, newConfig.EncryptionPrivateKey)
	})
}

func TestTopicExistsTrue(t *testing.T) {
	if os.Getenv("KAFKA_ON_LOCALHOST") != "true" {
		t.Skip("manual or integration test. Requires kafka")
	}
	broker := initializesMessageBroker(t)
	topic := broker.PluginConfig.SelfTopicName
	broker.CreateTopic(context.Background(), topic, nil) // nolint:errcheck

	r, err := broker.TopicExists(topic)
	require.NoError(t, err)

	require.Equal(t, true, r)
}

func TestTopicExistsFalse(t *testing.T) {
	if os.Getenv("KAFKA_ON_LOCALHOST") != "true" {
		t.Skip("manual or integration test. Requires kafka")
	}
	broker := initializesMessageBroker(t)
	topic := "newer_exists"

	r, err := broker.TopicExists(topic)
	require.NoError(t, err)

	require.Equal(t, false, r)
}

const serverKafka = "localhost:9094"

func initializesMessageBroker(t *testing.T) *MessageBroker {
	storage := &logical.InmemStorage{}
	key := "kafka.config"
	pl := "kafka.plugin.config"

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	config := BrokerConfig{
		Endpoints:            []string{serverKafka},
		EncryptionPrivateKey: pk,
		EncryptionPublicKey:  &pk.PublicKey,
	}

	plugin := PluginConfig{
		SelfTopicName: "topic_" + strings.ReplaceAll(time.Now().String()[11:23], ":", "_"),
	}
	d1, _ := json.Marshal(config)
	d2, _ := json.Marshal(plugin)
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: key, Value: d1})
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: pl, Value: d2})

	mb, err := NewMessageBroker(context.TODO(), storage, hclog.NewNullLogger())
	require.NoError(t, err)
	return mb
}
