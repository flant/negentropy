package kafka_destination

import (
	"crypto/rsa"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const (
	MetadataTopicType = "Metadata"
)

type MetadataKafkaDestination struct {
	commonDest
	mb *kafka.MessageBroker

	pubKey      *rsa.PublicKey
	topic       string
	replicaName string
}

func NewMetadataKafkaDestination(mb *kafka.MessageBroker, replica model.Replica) *MetadataKafkaDestination {
	return &MetadataKafkaDestination{
		commonDest:  newCommonDest(),
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
	if !mkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := mkd.simpleObjectKafker(mkd.topic, obj, mkd.mb.EncryptionPrivateKey(), mkd.pubKey, true)
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (mkd *MetadataKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := mkd.simpleObjectDeleteKafker(mkd.topic, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}

// TODO: fill all object types:
/*FeatureFlag
Role
Tenant
TenantFeatureFlag
User
ServiceAccount
Group
Project
ProjectFeatureFlag
IdentitySharing
RoleBinding
RoleBindingApproval
Tokens?
UserToken
ServiceAccountToken
*/
func (mkd *MetadataKafkaDestination) isValidObjectType(objType string) bool {
	switch objType {
	case model.UserType, model.TenantType:
		return true

	default:
		return false
	}
}
