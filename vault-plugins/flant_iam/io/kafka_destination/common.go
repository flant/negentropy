package kafka_destination

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func signData(data []byte, pk *rsa.PrivateKey) ([]byte, error) {
	signHash := sha256.Sum256(data)
	sign, err := rsa.SignPKCS1v15(rand.Reader, pk, crypto.SHA256, signHash[:])

	return sign, err
}

func encryptData(data []byte, pub *rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, data, nil)
}

func simpleObjectKafker(topic string, obj io.MemoryStorableObject, pk *rsa.PrivateKey, pub *rsa.PublicKey) (kafka.Message, error) {
	key := fmt.Sprintf("%s/%s", obj.ObjType(), obj.ObjId())
	data, err := obj.Marshal(true)
	if err != nil {
		return kafka.Message{}, err
	}
	sign, err := signData(data, pk)
	header := kafka.Header{Key: "signature", Value: sign}

	data, err = encryptData(data, pub)

	msg := kafka.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   data,
		Headers: []kafka.Header{header},
		Time:    time.Time{},
	}

	return msg, nil
}

func simpleObjectDeleteKafker(topic string, obj io.MemoryStorableObject, pk *rsa.PrivateKey) (kafka.Message, error) {
	key := fmt.Sprintf("%s/%s", obj.ObjType(), obj.ObjId())
	sign, err := signData(nil, pk)
	if err != nil {
		return kafka.Message{}, err
	}
	header := kafka.Header{Key: "signature", Value: sign}

	msg := kafka.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   nil,
		Headers: []kafka.Header{header},
		Time:    time.Time{},
	}

	return msg, nil
}
