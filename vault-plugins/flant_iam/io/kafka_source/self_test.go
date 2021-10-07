package kafka_source

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func TestInitial(t *testing.T) {
	if os.Getenv("KAFKA_ON_LOCALHOST") != "true" {
		t.Skip("manual or integration test. Requires kafka")
	}
	storage := &logical.InmemStorage{}
	key := "kafka.config"
	pl := "kafka.plugin.config"

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	topic := "topic_kafka_source_" + strings.ReplaceAll(time.Now().String()[11:23], ":", "_")

	config := kafka.BrokerConfig{
		Endpoints:             []string{"localhost:9093"},
		ConnectionPrivateKey:  nil,
		ConnectionCertificate: nil,
		EncryptionPrivateKey:  pk,
		EncryptionPublicKey:   &pk.PublicKey,
	}

	plugin := kafka.PluginConfig{
		SelfTopicName: topic,
	}
	d1, _ := json.Marshal(config)
	d2, _ := json.Marshal(plugin)
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: key, Value: d1})
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: pl, Value: d2})

	mb, err := kafka.NewMessageBroker(context.TODO(), storage, log.NewNullLogger())
	require.NoError(t, err)

	err = mb.CreateTopic(context.TODO(), topic, nil)
	require.NoError(t, err)

	ss := NewSelfKafkaSource(mb, []RestoreFunc{}, log.NewNullLogger())
	err = ss.Restore(nil)
	require.NoError(t, err)
}
