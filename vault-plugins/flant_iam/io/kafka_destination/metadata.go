package kafka_destination

import (
	"crypto/rsa"

	"github.com/hashicorp/go-memdb"
	"github.com/segmentio/kafka-go"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const (
	MetadataTopicType = "Metadata"
)

type MetadataKafkaDestination struct {
	mb *sharedkafka.MessageBroker

	pubKey      *rsa.PublicKey
	topic       string
	replicaName string
}

func NewMetadataKafkaDestination(mb *sharedkafka.MessageBroker, replica model.Replica) *MetadataKafkaDestination {
	return &MetadataKafkaDestination{
		mb:          mb,
		pubKey:      replica.PublicKey,
		topic:       "root_source." + replica.Name,
		replicaName: replica.Name,
	}
}

func (mkd *MetadataKafkaDestination) ReplicaName() string {
	return mkd.replicaName
}

func (mkd *MetadataKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.mb.Configured() {
		return nil, nil
	}
	msg, err := simpleObjectKafker(mkd.topic, obj, mkd.mb.EncryptionPrivateKey(), mkd.pubKey)
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil

}

func (mkd *MetadataKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.mb.Configured() {
		return nil, nil
	}
	msg, err := simpleObjectDeleteKafker(mkd.topic, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}
