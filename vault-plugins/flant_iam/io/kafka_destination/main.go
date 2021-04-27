package kafka_destination

import (
	"github.com/hashicorp/go-memdb"
	"github.com/segmentio/kafka-go"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type MainKafkaDestination struct {
	mb *sharedkafka.MessageBroker

	topic string
}

func NewMainKafkaDestination(mb *sharedkafka.MessageBroker, topic string) *MainKafkaDestination {
	return &MainKafkaDestination{
		mb:    mb,
		topic: topic,
	}
}

func (mkd *MainKafkaDestination) ReplicaName() string {
	return "root"
}

func (mkd *MainKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	msg, err := simpleObjectKafker(mkd.topic, obj, mkd.mb.EncryptionPrivateKey(), mkd.mb.EncryptionPublicKey())
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (mkd *MainKafkaDestination) ProcessObjectDelete(ms *io.MemoryStore, tnx *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	msg, err := simpleObjectDeleteKafker(mkd.topic, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}
