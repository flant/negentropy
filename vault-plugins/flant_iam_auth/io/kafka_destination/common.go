package kafka_destination

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

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
