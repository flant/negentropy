package kafka_destination

import (
	"crypto/rsa"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/explugin/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type IAMKafkaDestination struct {
	commonDest
	mb *kafka.MessageBroker

	pubKey        *rsa.PublicKey
	extensionName string
}

func NewIAMKafkaDestination(mb *kafka.MessageBroker, pubkey *rsa.PublicKey, extensionName string) *IAMKafkaDestination {
	return &IAMKafkaDestination{
		commonDest:    newCommonDest(),
		mb:            mb,
		pubKey:        pubkey,
		extensionName: extensionName,
	}
}

func (vkd *IAMKafkaDestination) topic() string {
	return fmt.Sprintf("extension.%s", vkd.extensionName)
}

func (vkd *IAMKafkaDestination) ReplicaName() string {
	return vkd.extensionName
}

func (vkd *IAMKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !vkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := vkd.simpleKafkaSender(vkd.topic(), obj, vkd.mb.EncryptionPrivateKey(), vkd.pubKey, true)
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (vkd *IAMKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !vkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := vkd.simpleKafkaDeleter(vkd.topic(), obj, vkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}

// TODO: (permanent) fill all object types for vault queue
func (vkd *IAMKafkaDestination) isValidObjectType(objType string) bool {
	switch objType {
	case model.UserType:
		return true

	default:
		return false
	}
}
