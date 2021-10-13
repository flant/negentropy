package kafka_destination

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type MultipassGenerationKafkaDestination struct {
	mb     *kafka.MessageBroker
	logger hclog.Logger
}

func NewMultipassGenerationKafkaDestination(mb *kafka.MessageBroker, parentLogger hclog.Logger) *MultipassGenerationKafkaDestination {
	return &MultipassGenerationKafkaDestination{
		mb:     mb,
		logger: parentLogger.Named("KafkaDestinationMultipassGen"),
	}
}

func (mkd *MultipassGenerationKafkaDestination) ReplicaName() string {
	return io.MultipassNumberGenerationTopic
}

func (mkd *MultipassGenerationKafkaDestination) ProcessObject(_ *sharedio.MemoryStore, _ *memdb.Txn, obj sharedio.MemoryStorableObject) ([]kafka.Message, error) {
	if obj.ObjType() != model.MultipassGenerationNumberType {
		return nil, nil
	}

	msg, err := mkd.sendObject(io.MultipassNumberGenerationTopic, obj, mkd.mb.EncryptionPrivateKey(), mkd.mb.EncryptionPublicKey())
	if err != nil {
		return nil, err
	}

	return []kafka.Message{msg}, nil
}

func (mkd *MultipassGenerationKafkaDestination) ProcessObjectDelete(_ *sharedio.MemoryStore, _ *memdb.Txn, obj sharedio.MemoryStorableObject) ([]kafka.Message, error) {
	if obj.ObjType() != model.MultipassGenerationNumberType {
		return nil, nil
	}

	msg, err := mkd.sendObjectTombstone(io.MultipassNumberGenerationTopic, obj, mkd.mb.EncryptionPrivateKey())
	if err != nil {
		return nil, err
	}
	return []kafka.Message{msg}, nil
}

func (mkd *MultipassGenerationKafkaDestination) signData(data []byte, pk *rsa.PrivateKey) ([]byte, error) {
	signHash := sha256.Sum256(data)
	sign, err := rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, signHash[:])

	return sign, err
}

func (mkd *MultipassGenerationKafkaDestination) sendObject(topic string, obj sharedio.MemoryStorableObject, pk *rsa.PrivateKey, pub *rsa.PublicKey) (kafka.Message, error) {
	key := fmt.Sprintf("%s/%s", obj.ObjType(), obj.ObjId())
	mkd.logger.Debug(fmt.Sprintf("key to send %s", key))
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

func (mkd *MultipassGenerationKafkaDestination) sendObjectTombstone(topic string, obj sharedio.MemoryStorableObject, pk *rsa.PrivateKey) (kafka.Message, error) {
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
