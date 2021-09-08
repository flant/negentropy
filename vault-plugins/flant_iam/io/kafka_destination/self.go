package kafka_destination

import (
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type SelfKafkaDestination struct {
	commonDest
	mb *kafka.MessageBroker
}

func NewSelfKafkaDestination(mb *kafka.MessageBroker) *SelfKafkaDestination {
	return &SelfKafkaDestination{
		commonDest: newCommonDest(),
		mb:         mb,
	}
}

func (mkd *SelfKafkaDestination) ReplicaName() string {
	return mkd.mb.PluginConfig.SelfTopicName
}

func (mkd *SelfKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	msg, err := mkd.simpleObjectKafker(
		mkd.mb.PluginConfig.SelfTopicName,
		obj,
		mkd.mb.EncryptionPrivateKey(), mkd.mb.EncryptionPublicKey(),
		true,
	)
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (mkd *SelfKafkaDestination) ProcessObjectDelete(ms *io.MemoryStore, txn *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	msg, err := mkd.simpleObjectDeleteKafker(mkd.mb.PluginConfig.SelfTopicName, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}
