package kafka_destination

import (
	"crypto/rsa"
	"fmt"

	"github.com/hashicorp/go-memdb"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
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

func NewVaultKafkaDestination(mb *kafka.MessageBroker, replica iam_model.Replica) *VaultKafkaDestination {
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

func (vkd *VaultKafkaDestination) isValidObjectType(objType string) bool {
	switch objType {
	case iam_model.TenantType,
		iam_model.ProjectType,
		iam_model.UserType,
		iam_model.ServiceAccountType,
		iam_model.RoleBindingType,
		iam_model.GroupType,
		iam_model.MultipassType,
		iam_model.ServiceAccountPasswordType,
		ext_model.ServerType:
		return true

	default:
		return false
	}
}
