package kafka_destination

import (
	"crypto/rsa"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const (
	VaultTopicType = "Vault"
)

type VaultKafkaDestination struct {
	commonDest
	mb *kafka.MessageBroker

	pubKey      *rsa.PublicKey
	replicaName string
}

func NewVaultKafkaDestination(mb *kafka.MessageBroker, replica model.Replica) *VaultKafkaDestination {
	return &VaultKafkaDestination{
		commonDest:  newCommonDest(),
		mb:          mb,
		pubKey:      replica.PublicKey,
		replicaName: replica.Name,
	}
}

func (vkd *VaultKafkaDestination) topic() string {
	return fmt.Sprintf("%s.%s", vkd.mb.PluginConfig.SelfTopicName, vkd.replicaName)
}

func (vkd *VaultKafkaDestination) ReplicaName() string {
	return vkd.replicaName
}

func (vkd *VaultKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !vkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := vkd.simpleObjectKafker(vkd.topic(), obj, vkd.mb.EncryptionPrivateKey(), vkd.pubKey, true)
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (vkd *VaultKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !vkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := vkd.simpleObjectDeleteKafker(vkd.topic(), obj, vkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}

// TODO: (permanent) fill all object types for vault queue
func (vkd *VaultKafkaDestination) isValidObjectType(objType string) bool {
	switch objType {
	// TODO Multipass (with sensitive data)
	// TODO ServiceAccountPassword (with sensitive data)
	case model.TenantType,
		model.ProjectType,
		model.UserType,
		model.ServiceAccountType,
		model.RoleBindingType,
		model.GroupType:
		return true

	default:
		return false
	}
}
