package kafka_destination

import (
	"crypto/rsa"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const (
	MetadataTopicType = "Metadata"
)

type MetadataKafkaDestination struct {
	commonDest
	mb *kafka.MessageBroker

	pubKey      *rsa.PublicKey
	replicaName string
}

func NewMetadataKafkaDestination(mb *kafka.MessageBroker, replica model.Replica) *MetadataKafkaDestination {
	return &MetadataKafkaDestination{
		commonDest:  newCommonDest(),
		mb:          mb,
		pubKey:      replica.PublicKey,
		replicaName: replica.Name,
	}
}

func (mkd *MetadataKafkaDestination) ReplicaName() string {
	return mkd.replicaName
}

func (mkd *MetadataKafkaDestination) topic() string {
	return fmt.Sprintf("%s.%s", mkd.mb.PluginConfig.SelfTopicName, mkd.replicaName)
}

func (mkd *MetadataKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := mkd.simpleObjectKafker(mkd.topic(), obj, mkd.mb.EncryptionPrivateKey(), mkd.pubKey, true)
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (mkd *MetadataKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := mkd.simpleObjectDeleteKafker(mkd.topic(), obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}

// TODO: (permanent) fill all object types for metadata queue
func (mkd *MetadataKafkaDestination) isValidObjectType(objType string) bool {
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
		model.ServiceAccountPasswordType:
		return true

	default:
		return false
	}
}
