package kafka_source

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-memdb"

	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/handlers/iam"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type RootKafkaSource struct {
	kf        *sharedkafka.MessageBroker
	decryptor *sharedkafka.Encrypter

	stopC chan struct{}
}

func NewRootKafkaSource(kf *sharedkafka.MessageBroker) *RootKafkaSource {
	return &RootKafkaSource{
		kf:        kf,
		decryptor: sharedkafka.NewEncrypter(),

		stopC: make(chan struct{}),
	}
}

func (rk *RootKafkaSource) Name() string {
	return rk.kf.PluginConfig.RootTopicName
}

func (rk *RootKafkaSource) Restore(txn *memdb.Txn) error {
	rootTopic := rk.kf.PluginConfig.RootTopicName
	replicaName := rk.kf.PluginConfig.SelfTopicName

	runConsumer := rk.kf.GetConsumer(replicaName, rootTopic, false)
	defer runConsumer.Close()

	r := rk.kf.GetRestorationReader(rootTopic)
	defer r.Close()

	return sharedkafka.RunRestorationLoop(r, runConsumer, replicaName, txn, rk.restoreMsgHandler)
}

func (rk *RootKafkaSource) restoreMsgHandler(txn *memdb.Txn, msg *kafka.Message) error {
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		log.Printf("wrong object Key format: %s\n", msg.Key)
		return fmt.Errorf("key has wong format: %s", msg.Key)
	}

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
		return fmt.Errorf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	if len(signature) == 0 {
		return fmt.Errorf("no signature found. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	err = rk.verifySign(signature, decrypted)
	if err != nil {
		return fmt.Errorf("wrong signature. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	// TODO: need huge switch-case here, with object Unmarshaling
	var inputObject interface{}
	switch splitted[0] {
	case iam_model.UserType:
		inputObject = &iam_model.User{}

	case iam_model.TenantType:
		inputObject = &iam_model.Tenant{}

	case iam_model.ProjectType:
		inputObject = &iam_model.Project{}

	case iam_model.ServerType:
		inputObject = &iam_model.Server{}

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

	return nil
}

func (rk *RootKafkaSource) Run(store *io.MemoryStore) {
	rootTopic := rk.kf.PluginConfig.RootTopicName
	replicaName := rk.kf.PluginConfig.SelfTopicName
	rd := rk.kf.GetConsumer(replicaName, rootTopic, false)

	sharedkafka.RunMessageLoop(rd, rk.msgHandler(store), rk.stopC)
}

func (rk *RootKafkaSource) msgHandler(store *io.MemoryStore) func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	return func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			// return fmt.Errorf("key has wong format: %s", string(msg.Key))
			return
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
			return
		}

		if len(signature) == 0 {
			log.Printf("no signature found. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			return
		}

		err = rk.verifySign(signature, decrypted)
		if err != nil {
			log.Printf("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			return
		}

		source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
		if err != nil {
			log.Println("build source message failed", err)
			return
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

func (rk *RootKafkaSource) processMessage(source *sharedkafka.SourceInputMessage, store *io.MemoryStore, objType, objID string, data []byte) error {
	tx := store.Txn(true)
	defer tx.Abort()

	err := iam.HandleNewMessageIamRootSource(tx, iam.NewObjectHandler(tx), objType, objID, data)
	if err != nil {
		return backoff.Permanent(err)
	}

	return tx.Commit(source)
}

func (rk *RootKafkaSource) Stop() {
	rk.stopC <- struct{}{}
}
