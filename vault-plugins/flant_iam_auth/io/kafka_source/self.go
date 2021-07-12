package kafka_source

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"log"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/self"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type SelfKafkaSource struct {
	kf        *sharedkafka.MessageBroker
	decryptor *sharedkafka.Encrypter

	api *vault.VaultEntityDownstreamApi

	stopC chan struct{}
}

func NewSelfKafkaSource(kf *sharedkafka.MessageBroker, api *vault.VaultEntityDownstreamApi) *SelfKafkaSource {
	return &SelfKafkaSource{
		kf:        kf,
		decryptor: sharedkafka.NewEncrypter(),
		api:       api,

		stopC: make(chan struct{}),
	}
}

func (sks *SelfKafkaSource) Name() string {
	return sks.kf.PluginConfig.SelfTopicName
}

func (sks *SelfKafkaSource) Restore(txn *memdb.Txn) error {
	r := sks.kf.GetRestorationReader(sks.kf.PluginConfig.SelfTopicName)
	defer r.Close()

	replicaName := sks.kf.PluginConfig.SelfTopicName

	runConsumer := sks.kf.GetConsumer(replicaName, replicaName, false)
	_ = runConsumer.Close()

	return sharedkafka.RunRestorationLoop(r, runConsumer, replicaName, txn, sks.restoreMsHandler)
}

func (sks *SelfKafkaSource) restoreMsHandler(txn *memdb.Txn, msg *kafka.Message) error {
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		return fmt.Errorf("key has wong format: %s", string(msg.Key))
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

	decrypted, err := sks.decryptData(msg.Value, chunked)
	if err != nil {
		return fmt.Errorf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	if len(signature) == 0 {
		return fmt.Errorf("no signature found. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	err = sks.verifySign(signature, decrypted)
	if err != nil {
		return fmt.Errorf("wrong signature. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	err = self.HandleRestoreMessagesSelfSource(txn, splitted[0], decrypted)
	if err != nil {
		return err
	}

	return nil
}

func (sks *SelfKafkaSource) Run(store *io.MemoryStore) {
	replicaName := sks.kf.PluginConfig.SelfTopicName
	rd := sks.kf.GetConsumer(replicaName, replicaName, false)

	sharedkafka.RunMessageLoop(rd, sks.messageHandler(store), sks.stopC)
}

func (sks *SelfKafkaSource) messageHandler(store *io.MemoryStore) func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	return func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			// TODO: ??
			// return fmt.Errorf("key has wong format: %s", string(msg.Key))
			return
		}
		objType := splitted[0]
		objId := splitted[1]

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

		decrypted, err := sks.decryptData(msg.Value, chunked)
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

		err = sks.verifySign(signature, decrypted)
		if err != nil {
			log.Printf("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			return
		}

		source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
		if err != nil {
			log.Println("build source message failed", err)
		}

		operation := func() error {
			msgDecoded := &sharedkafka.MsgDecoded{
				Type: objType,
				ID:   objId,
				Data: decrypted,
			}
			return sks.processMessage(source, store, msgDecoded)
		}
		err = backoff.Retry(operation, backoff.NewExponentialBackOff())
		if err != nil {
			panic(err)
		}
	}
}

func (sks *SelfKafkaSource) processMessage(source *sharedkafka.SourceInputMessage, store *io.MemoryStore, msg *sharedkafka.MsgDecoded) error {
	tx := store.Txn(true)
	defer tx.Abort()

	err := self.HandleNewMessageSelfSource(tx, self.NewObjectHandler(store, tx, sks.api), msg)
	if err != nil {
		return err
	}
	return tx.Commit(source)
}

func (sks *SelfKafkaSource) decryptData(data []byte, chunked bool) ([]byte, error) {
	return sks.decryptor.Decrypt(data, sks.kf.EncryptionPrivateKey(), chunked)
}

func (sks *SelfKafkaSource) verifySign(signature []byte, data []byte) error {
	hashed := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(sks.kf.EncryptionPublicKey(), crypto.SHA256, hashed[:], signature)
}

func (sks *SelfKafkaSource) Stop() {
	sks.stopC <- struct{}{}
}
