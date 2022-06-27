package kafka_destination

import (
	"crypto/rsa"
	"fmt"

	ext_ff_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	ext_sa_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
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

func (vkd *VaultKafkaDestination) isValidObjectType(objType string) bool {
	switch objType {
	case model.TenantType,
		model.ProjectType,
		model.UserType,
		model.ServiceAccountType,
		model.FeatureFlagType,
		model.RoleType,
		model.RoleBindingType,
		model.RoleBindingApprovalType,
		model.IdentitySharingType,
		model.GroupType,
		model.MultipassType,
		model.ServiceAccountPasswordType,

		ext_sa_model.ServerType,

		ext_ff_model.TeamType,
		ext_ff_model.TeammateType,
		ext_ff_model.ServicePackType:
		return true

	default:
		return false
	}
}
