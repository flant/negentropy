package io

import (
	"context"
	"encoding/binary"
	"os"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func Test_storeLastOffset(t *testing.T) {
	ctx := context.TODO()
	storage := &logical.InmemStorage{}
	pt := "plugin-self-topic-name"
	tp := "topic-name"
	expected := int64(123)

	err := StoreLastOffsetToStorage(ctx, storage, pt, tp, 123)

	require.NoError(t, err)
	se, err := storage.Get(ctx, lastOffsetStorageKey(pt, tp))
	require.NoError(t, err)
	require.NotNil(t, se)
	require.Len(t, se.Value, 8)
	actual := int64(binary.BigEndian.Uint64(se.Value))
	require.Equal(t, expected, actual)
}

func Test_getLastOffsetFromStorage(t *testing.T) {
	ctx := context.TODO()
	storage := &logical.InmemStorage{}
	pt := "plugin-self-topic-name"
	tp := "topic-name"
	expected := int64(123)
	err := StoreLastOffsetToStorage(ctx, storage, pt, tp, 123)
	require.NoError(t, err)

	actual, err := LastOffsetFromStorage(ctx, storage, pt, tp)

	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func Test_Test_getLastOffsetFromEmptyStorage(t *testing.T) {
	ctx := context.TODO()
	storage := &logical.InmemStorage{}
	pt := "plugin-self-topic-name"
	tp := "topic-name"

	actual, err := LastOffsetFromStorage(ctx, storage, pt, tp)

	require.NoError(t, err)
	require.Equal(t, int64(-1), actual)
}

func TestLastOffsetByNewConsumer(t *testing.T) {
	if os.Getenv("KAFKA_ON_LOCALHOST") != "true" {
		t.Skip("manual or integration test. Requires kafka")
	}
	sharedkafka.DoNotEncrypt = true // TODO remove
	broker := initializesMessageBroker(t)
	topic := broker.PluginConfig.SelfTopicName
	err := broker.CreateTopic(context.TODO(), topic, nil)
	assert.NoError(t, err, "creating topic")
	fillTopic(t, broker, 15)

	newConsumer, err := broker.GetRestorationReader()
	assert.NoError(t, err, "getting restoration reader")
	l, p, err := LastOffsetByNewConsumer(newConsumer, topic)

	require.NoError(t, err, "LastOffsetByNewConsumer")
	assert.Equal(t, int64(14), l)
	assert.Equal(t, int32(0), p)
}

func TestLastOffsetByNewConsumerEmptyTopic(t *testing.T) {
	if os.Getenv("KAFKA_ON_LOCALHOST") != "true" {
		t.Skip("manual or integration test. Requires kafka")
	}
	sharedkafka.DoNotEncrypt = true // TODO remove
	broker := initializesMessageBroker(t)
	topic := broker.PluginConfig.SelfTopicName
	err := broker.CreateTopic(context.TODO(), topic, nil)
	assert.NoError(t, err, "creating topic")

	newConsumer, err := broker.GetRestorationReader()
	assert.NoError(t, err, "getting restoration reader")
	l, p, err := LastOffsetByNewConsumer(newConsumer, topic)

	require.NoError(t, err, "LastOffsetByNewConsumer")
	assert.Equal(t, int64(0), l)
	assert.Equal(t, int32(0), p)
}
