package kafka_destination

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type SelfKafkaDestination struct {
	mb        *kafka.MessageBroker
	encrypter *kafka.Encrypter
}

func NewSelfKafkaDestination(mb *kafka.MessageBroker) *SelfKafkaDestination {
	return &SelfKafkaDestination{
		mb:        mb,
		encrypter: kafka.NewEncrypter(),
	}
}

func (mkd *SelfKafkaDestination) ReplicaName() string {
	return mkd.mb.PluginConfig.SelfTopicName
}

func (mkd *SelfKafkaDestination) ProcessObject(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := mkd.simpleObjectKafker(mkd.mb.PluginConfig.SelfTopicName, obj, mkd.mb.EncryptionPrivateKey(), mkd.mb.EncryptionPublicKey())
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (mkd *SelfKafkaDestination) ProcessObjectDelete(_ *io.MemoryStore, _ *memdb.Txn, obj io.MemoryStorableObject) ([]kafka.Message, error) {
	if !mkd.isValidObjectType(obj.ObjType()) {
		return nil, nil
	}
	msg, err := mkd.simpleObjectDeleteKafker(mkd.mb.PluginConfig.SelfTopicName, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}

// only models from this plugin
func (mkd *SelfKafkaDestination) isValidObjectType(objType string) bool {
	if jwtkafka.WriteInSelfQueue(objType) {
		return true
	}

	switch objType {
	case model.EntityType,
		model.AuthSourceType,
		model.EntityAliasType,
		model.AuthMethodType,
		model.JWTIssueTypeType:
		return true
	}

	return false
}

func (mkd *SelfKafkaDestination) signData(data []byte, pk *rsa.PrivateKey) ([]byte, error) {
	signHash := sha256.Sum256(data)
	sign, err := rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, signHash[:])

	return sign, err
}

func (mkd *SelfKafkaDestination) encryptData(data []byte, pub *rsa.PublicKey) ([]byte, bool, error) {
	return mkd.encrypter.Encrypt(data, pub)
}

func (mkd *SelfKafkaDestination) simpleObjectKafker(topic string, obj io.MemoryStorableObject,
	pk *rsa.PrivateKey, pub *rsa.PublicKey) (kafka.Message, error) {
	key := fmt.Sprintf("%s/%s", obj.ObjType(), obj.ObjId())
	data, err := json.Marshal(obj)
	if err != nil {
		return kafka.Message{}, err
	}
	sign, err := mkd.signData(data, pk)
	if err != nil {
		return kafka.Message{}, err
	}

	var chunked bool
	data, chunked, err = mkd.encryptData(data, pub)
	if err != nil {
		return kafka.Message{}, err
	}

	msg := kafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   data,
		Headers: map[string][]byte{"signature": sign},
	}

	if chunked {
		msg.Headers["chunked"] = []byte("true")
	}

	return msg, nil
}

func (mkd *SelfKafkaDestination) simpleObjectDeleteKafker(topic string, obj io.MemoryStorableObject, pk *rsa.PrivateKey) (kafka.Message, error) {
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
