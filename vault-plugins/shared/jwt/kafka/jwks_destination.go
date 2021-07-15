package kafka

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type JWKSKafkaDestination struct {
	mb *kafka.MessageBroker
}

func NewJWKSKafkaDestination(mb *kafka.MessageBroker) *JWKSKafkaDestination {
	return &JWKSKafkaDestination{
		mb: mb,
	}
}

func (mkd *JWKSKafkaDestination) ReplicaName() string {
	return topicName
}

func (mkd *JWKSKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if obj.ObjType() != jwt.JWKSType {
		return nil, nil
	}

	msg, err := mkd.sendObject(mkd.mb.PluginConfig.SelfTopicName, obj, mkd.mb.EncryptionPrivateKey(), mkd.mb.EncryptionPublicKey())
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (mkd *JWKSKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if obj.ObjType() != jwt.JWKSType {
		return nil, nil
	}

	msg, err := mkd.sendObjectTombstone(mkd.mb.PluginConfig.SelfTopicName, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}

func (mkd *JWKSKafkaDestination) signData(data []byte, pk *rsa.PrivateKey) ([]byte, error) {
	signHash := sha256.Sum256(data)
	sign, err := rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, signHash[:])

	return sign, err
}

func (mkd *JWKSKafkaDestination) sendObject(topic string, obj io.MemoryStorableObject, pk *rsa.PrivateKey, pub *rsa.PublicKey) (kafka.Message, error) {
	key := fmt.Sprintf("%s/%s", obj.ObjType(), obj.ObjId())
	data, err := json.Marshal(obj)
	if err != nil {
		return kafka.Message{}, err
	}
	sign, err := mkd.signData(data, pk)
	if err != nil {
		return kafka.Message{}, err
	}

	msg := kafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   data,
		Headers: map[string][]byte{"signature": sign},
	}

	return msg, nil
}

func (mkd *JWKSKafkaDestination) sendObjectTombstone(topic string, obj io.MemoryStorableObject, pk *rsa.PrivateKey) (kafka.Message, error) {
	key := fmt.Sprintf("%s/%s", obj.ObjType(), obj.ObjId())
	sign, err := mkd.signData(nil, pk)
	if err != nil {
		return kafka.Message{}, err
	}

	msg := kafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   nil,
		Headers: map[string][]byte{"signature": sign},
	}

	return msg, nil
}
