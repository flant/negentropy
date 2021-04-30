package kafka_destination

import (
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type SelfKafkaDestination struct {
	mb        *kafka.MessageBroker
	encrypter *kafka.Encrypter
	topic     string
}

func NewSelfKafkaDestination(mb *kafka.MessageBroker) *SelfKafkaDestination {
	return &SelfKafkaDestination{
		mb:        mb,
		encrypter: kafka.NewEncrypter(),
		topic:     mb.PluginConfig.SelfTopicName,
	}
}

func (mkd *SelfKafkaDestination) ReplicaName() string {
	return mkd.mb.PluginConfig.SelfTopicName
}

func (mkd *SelfKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := mkd.simpleObjectKafker(mkd.topic, obj, mkd.mb.EncryptionPrivateKey(), mkd.mb.EncryptionPublicKey())
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (mkd *SelfKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := mkd.simpleObjectDeleteKafker(mkd.topic, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}

// only models from this plugin
func (mkd *SelfKafkaDestination) isValidObjectType(objType string) bool {
	switch objType {
	case model.EntityType, model.PendingLoginType:
		return true

	default:
		return false
	}
}
