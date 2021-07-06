package kafka_destination

import (
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type SelfKafkaDestination struct {
	commonDest
	mb *sharedkafka.MessageBroker
}

func NewSelfKafkaDestination(mb *sharedkafka.MessageBroker) *SelfKafkaDestination {
	return &SelfKafkaDestination{
		commonDest: newCommonDest(),
		mb:         mb,
	}
}

func (mkd *SelfKafkaDestination) ReplicaName() string {
	return mkd.mb.PluginConfig.SelfTopicName
}

func (mkd *SelfKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]sharedkafka.Message, error) {
	msg, err := mkd.simpleKafkaSender(
		mkd.mb.PluginConfig.SelfTopicName,
		obj,
		mkd.mb.EncryptionPrivateKey(), mkd.mb.EncryptionPublicKey(),
		true,
	)
	if err != nil {
		return nil, err
	}

	return []sharedkafka.Message{msg}, nil
}

func (mkd *SelfKafkaDestination) ProcessObjectDelete(ms *io.MemoryStore, tnx *memdb.Txn, obj io.MemoryStorableObject) ([]sharedkafka.Message, error) {
	msg, err := mkd.simpleKafkaDeleter(mkd.mb.PluginConfig.SelfTopicName, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []sharedkafka.Message{msg}, nil
}
