package kafka_source

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/self"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type SelfSourceMsgHandlerFactory func(store *io.MemoryStore, tx *io.MemoryStoreTxn) self.ModelHandler

type SelfKafkaSource struct {
	kf        *sharedkafka.MessageBroker
	decryptor *sharedkafka.Encrypter

	handlerFactory SelfSourceMsgHandlerFactory
	logger hclog.Logger

	stopC chan struct{}
}

func NewSelfKafkaSource(kf *sharedkafka.MessageBroker, handlerFactory SelfSourceMsgHandlerFactory, logger hclog.Logger) *SelfKafkaSource {
	return &SelfKafkaSource{
		kf:        kf,
		decryptor: sharedkafka.NewEncrypter(),
		handlerFactory:       handlerFactory,

		stopC: make(chan struct{}),

		logger: logger,
	}
}

func (sks *SelfKafkaSource) Name() string {
	return sks.kf.PluginConfig.SelfTopicName
}

func (sks *SelfKafkaSource) Restore(txn *memdb.Txn) error {
	replicaName := sks.kf.PluginConfig.SelfTopicName
	sks.logger.Error("Restore start",  "replica name", replicaName)
	defer sks.logger.Error("Restore end",  "replica name", replicaName)


	r := sks.kf.GetRestorationReader(sks.kf.PluginConfig.SelfTopicName)
	defer r.Close()


	runConsumer := sks.kf.GetConsumer(replicaName, replicaName, false)
	defer runConsumer.Close()

	sks.logger.Error("Start restoration self got run consumer")

	return sharedkafka.RunRestorationLoop(r, runConsumer, replicaName, txn, sks.restoreMsHandler)
}

func (sks *SelfKafkaSource) restoreMsHandler(txn *memdb.Txn, msg *kafka.Message) error {
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		return fmt.Errorf("key has wong format: %s", string(msg.Key))
	}

	sks.logger.Error("Restore - keys", "keys", splitted)

	var signature []byte
	var chunked bool

	sks.logger.Error("Restore - Start parse header")
	for _, header := range msg.Headers {
		sks.logger.Error("Restore - Switch header", "header", header)
		switch header.Key {
		case "signature":
			signature = header.Value

		case "chunked":
			chunked = true
		}
	}

	sks.logger.Error("Restore - Start decrypt message", "msg",  msg.Value)
	decrypted, err := sks.decryptData(msg.Value, chunked)
	if err != nil {
		return fmt.Errorf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	sks.logger.Error("Restore - Message decrypted", "decrypted", decrypted)

	if len(signature) == 0 {
		return fmt.Errorf("no signature found. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	err = sks.verifySign(signature, decrypted)
	if err != nil {
		return fmt.Errorf("wrong signature. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	sks.logger.Error("Restore - Message verified", "decrypted", decrypted)

	err = self.HandleRestoreMessagesSelfSource(txn, splitted[0], decrypted)
	if err != nil {
		return err
	}

	return nil
}

func (sks *SelfKafkaSource) Run(store *io.MemoryStore) {
	replicaName := sks.kf.PluginConfig.SelfTopicName
	sks.logger.Error("Watcher - start", "replica_name", replicaName)
	defer func() {
		sks.logger.Error("Watcher - stop", "replica_name", replicaName)
	}()

	rd := sks.kf.GetConsumer(replicaName, replicaName, false)
	sks.logger.Error("Watcher - got consumer", "replica_name", replicaName)

	sharedkafka.RunMessageLoop(rd, sks.messageHandler(store), sks.stopC, sks.logger)
}

func (sks *SelfKafkaSource) messageHandler(store *io.MemoryStore) func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	return func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
		panic("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
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
			    sks.logger.Error("can't decrypt message. Skipping: %s in topic: %s at offset %d\n",
					msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			return
		}

		if len(signature) == 0 {
			sks.logger.Error("no signature found. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			return
		}

		err = sks.verifySign(signature, decrypted)
		if err != nil {
			sks.logger.Error("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			return
		}

		source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
		if err != nil {
			sks.logger.Error("build source message failed", err)
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

	sks.logger.Error("Handle new message", msg.ID, msg.Type)

	panic("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

	err := self.HandleNewMessageSelfSource(tx, sks.handlerFactory(store, tx), msg)
	if err != nil {
		sks.logger.Error("Error handle message message", msg.ID, msg.Type, err)
		return err
	}

	sks.logger.Error("Message processed successfully", msg.ID, msg.Type)

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
