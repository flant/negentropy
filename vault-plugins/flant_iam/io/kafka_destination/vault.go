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
	VaultTopicType = "Vault"
)

type VaultKafkaDestination struct {
	mb *sharedkafka.MessageBroker

	pubKey      *rsa.PublicKey
	topic       string
	replicaName string
}

func NewVaultKafkaDestination(mb *sharedkafka.MessageBroker, replica model.Replica) *VaultKafkaDestination {
	return &VaultKafkaDestination{
		mb:          mb,
		pubKey:      replica.PublicKey,
		topic:       "root_source." + replica.Name,
		replicaName: replica.Name,
	}
}

func (vkd *VaultKafkaDestination) ReplicaName() string {
	return vkd.replicaName
}

func (vkd *VaultKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !vkd.mb.Configured() {
		return nil, nil
	}
	msg, err := simpleObjectKafker(vkd.topic, obj, vkd.mb.EncryptionPrivateKey(), vkd.pubKey)
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil

}

func (vkd *VaultKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !vkd.mb.Configured() {
		return nil, nil
	}
	msg, err := simpleObjectDeleteKafker(vkd.topic, obj, vkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}
