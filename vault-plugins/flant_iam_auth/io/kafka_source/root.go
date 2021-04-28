package kafka_source

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/handlers/iam"
	"log"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type RootKafkaSource struct {
	kf        *kafka.MessageBroker
	decryptor *kafka.Encrypter
}

func NewRootKafkaSource(kf *kafka.MessageBroker) *RootKafkaSource {
	return &RootKafkaSource{
		kf:        kf,
		decryptor: kafka.NewEncrypter(),
	}
}

func (rk *RootKafkaSource) Restore(txn *memdb.Txn) error {
	rootTopic := rk.kf.PluginConfig.RootTopicName
	replicaName := rk.kf.PluginConfig.SelfTopicName
	runConsumer := rk.kf.GetConsumer(replicaName, rootTopic, false)
	_, lastOffset, err := runConsumer.GetWatermarkOffsets(replicaName, 0)
	if err != nil {
		return err
	}
	_ = runConsumer.Close()

	if lastOffset <= 0 {
		return nil
	}

	r := rk.kf.GetRestorationReader(rootTopic)
	defer r.Close()

	for {
		m, err := r.ReadMessage(-1)
		if err != nil {
			return err
		}

		splitted := strings.Split(string(m.Key), "/")
		if len(splitted) != 2 {
			return fmt.Errorf("key has wong format: %s", string(m.Key))
		}

		var signature []byte
		var chunked bool
		for _, header := range m.Headers {
			switch header.Key {
			case "signature":
				signature = header.Value

			case "chunked":
				chunked = true
			}
		}

		decrypted, err := rk.decryptData(m.Value, chunked)
		if err != nil {
			log.Printf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n", m.Key, *m.TopicPartition.Topic, m.TopicPartition.Offset)
			continue
		}

		if len(signature) == 0 {
			log.Printf("no signature found. Skipping message: %s in topic: %s at offset %d\n", m.Key, *m.TopicPartition.Topic, m.TopicPartition.Offset)
			continue
		}

		err = rk.verifySign(signature, decrypted)
		if err != nil {
			log.Printf("wrong signature. Skipping message: %s in topic: %s at offset %d\n", m.Key, *m.TopicPartition.Topic, m.TopicPartition.Offset)
			continue
		}

		// TODO: need huge switch-case here, with object Unmarshaling
		var inputObject interface{}
		switch splitted[0] {
		case model.UserType:
			inputObject = &model.User{}

		default:
			return errors.New("is not implemented yet")
		}
		err = json.Unmarshal(decrypted, inputObject)
		if err != nil {
			return err
		}

		err = txn.Insert(splitted[0], inputObject)
		if err != nil {
			return err
		}

		if int64(m.TopicPartition.Offset) == lastOffset-1 {
			return nil
		}
	}
}

func (rk *RootKafkaSource) Run(store *io.MemoryStore) {
	rootTopic := rk.kf.PluginConfig.RootTopicName
	replicaName := rk.kf.PluginConfig.SelfTopicName
	rd := rk.kf.GetConsumer(replicaName, rootTopic, false)

	for {
		msg, err := rd.ReadMessage(-1)
		if err != nil {
			log.Println("Error reading message", err)
			continue // TODO: what to do?
		}

		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			// return fmt.Errorf("key has wong format: %s", string(msg.Key))
			continue
		}
		objType, objID := splitted[0], splitted[1]

		var signature []byte
		var chunked bool
		for _, header := range msg.Headers {
			switch header.Key {
			case "signature":
				signature = header.Value

			case "chunked":
				chunked = true
			}
		}

		decrypted, err := rk.decryptData(msg.Value, chunked)
		if err != nil {
			log.Printf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			continue
		}

		if len(signature) == 0 {
			log.Printf("no signature found. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			continue
		}

		err = rk.verifySign(signature, decrypted)
		if err != nil {
			log.Printf("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			continue
		}

		source, err := kafka.NewSourceInputMessage(rd, msg.TopicPartition)
		if err != nil {
			log.Println("build source message failed", err)
		}

		operation := func() error {
			return rk.processMessage(source, store, objType, objID, decrypted)
		}
		err = backoff.Retry(operation, backoff.NewExponentialBackOff())
		if err != nil {
			panic(err)
		}
	}
}

func (rk *RootKafkaSource) decryptData(data []byte, chunked bool) ([]byte, error) {
	return rk.decryptor.Decrypt(data, rk.kf.EncryptionPrivateKey(), chunked)
}

func (rk *RootKafkaSource) verifySign(signature []byte, data []byte) error {
	hashed := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(rk.kf.PluginConfig.RootPublicKey, crypto.SHA256, hashed[:], signature)
}

func (rk *RootKafkaSource) processMessage(source *kafka.SourceInputMessage, store *io.MemoryStore, objType, objID string, data []byte) error {
	tx := store.Txn(true)
	defer tx.Abort()

	err := iam.HandleNewMessageIamRootSource(tx, iam.NewObjectHandler(tx), objType, objID, data)
	if err != nil {
		return backoff.Permanent(err)
	}

	return tx.Commit(source)
}
