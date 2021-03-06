package kafka_destination

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type commonDest struct {
	encrypter *kafka.Encrypter
}

func newCommonDest() commonDest {
	return commonDest{encrypter: kafka.NewEncrypter()}
}

func (cd *commonDest) signData(data []byte, pk *rsa.PrivateKey) ([]byte, error) {
	signHash := sha256.Sum256(data)
	sign, err := rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, signHash[:])

	return sign, err
}

func (cd *commonDest) encryptData(data []byte, pub *rsa.PublicKey) ([]byte, bool, error) {
	return cd.encrypter.Encrypt(data, pub)
}

func (cd *commonDest) simpleObjectKafker(topic string, obj io.MemoryStorableObject, pk *rsa.PrivateKey, pub *rsa.PublicKey, includeSensitive bool) (kafka.Message, error) {
	key := fmt.Sprintf("%s/%s", obj.ObjType(), obj.ObjId())

	var public interface{} = obj
	if !includeSensitive {
		public = iam_repo.OmitSensitive(obj)
	}
	data, err := json.Marshal(public)
	if err != nil {
		return kafka.Message{}, err
	}
	sign, err := cd.signData(data, pk)
	if err != nil {
		return kafka.Message{}, err
	}
	var chunked bool
	data, chunked, err = cd.encryptData(data, pub)
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

func (cd *commonDest) simpleObjectDeleteKafker(topic string, obj io.MemoryStorableObject, pk *rsa.PrivateKey) (kafka.Message, error) {
	key := fmt.Sprintf("%s/%s", obj.ObjType(), obj.ObjId())
	sign, err := cd.signData(nil, pk)
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
