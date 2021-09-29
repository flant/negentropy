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
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type SelfSourceMsgHandlerFactory func(store *io.MemoryStore, tx *io.MemoryStoreTxn) self.ModelHandler

type SelfKafkaSource struct {
	kf        *sharedkafka.MessageBroker
	decryptor *sharedkafka.Encrypter

	handlerFactory SelfSourceMsgHandlerFactory
	logger         hclog.Logger

	stopC chan struct{}
}

func NewSelfKafkaSource(kf *sharedkafka.MessageBroker, handlerFactory SelfSourceMsgHandlerFactory, logger hclog.Logger) *SelfKafkaSource {
	return &SelfKafkaSource{
		kf:             kf,
		decryptor:      sharedkafka.NewEncrypter(),
		handlerFactory: handlerFactory,

		stopC: make(chan struct{}),

		logger: logger,
	}
}

func (sks *SelfKafkaSource) Name() string {
	return sks.kf.PluginConfig.SelfTopicName
}

func (sks *SelfKafkaSource) Restore(txn *memdb.Txn, _ hclog.Logger) error {
	replicaName := sks.kf.PluginConfig.SelfTopicName
	sks.logger.Debug("Restore - start", "replica name", replicaName)
	defer sks.logger.Debug("Restore - end", "replica name", replicaName)

	restorationConsumer := sks.kf.GetRestorationReader()
	defer restorationConsumer.Close()

	// groupId := fmt.Sprintf("restore-%s", replicaName)
	groupId := replicaName
	runConsumer := sks.kf.GetUnsubscribedRunConsumer(groupId)
	defer runConsumer.Close()

	sks.logger.Debug(fmt.Sprintf("Restore - got consumer %s/%s/%s", groupId, replicaName, replicaName))

	return sharedkafka.RunRestorationLoop(restorationConsumer, runConsumer, replicaName, txn, sks.restoreMsHandler, sks.logger)
}

func (sks *SelfKafkaSource) restoreMsHandler(txn *memdb.Txn, msg *kafka.Message, _ hclog.Logger) error {
	l := sks.logger
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		return fmt.Errorf("key has wong format: %s", string(msg.Key))
	}

	sks.logger.Debug("Restore - keys", "keys", splitted)

	var signature []byte
	var chunked bool

	l.Debug("Restore - Start parse header")
	for _, header := range msg.Headers {
		l.Debug("Restore - Switch header", "header", header)
		switch header.Key {
		case "signature":
			signature = header.Value

		case "chunked":
			chunked = true
		}
	}

	l.Debug("Restore - Start decrypt message", "msg", msg.Value)
	var decrypted []byte
	if len(msg.Value) > 0 {
		var err error
		decrypted, err = sks.decryptData(msg.Value, chunked)
		if err != nil {
			return fmt.Errorf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
		}
	} else {
		l.Debug(fmt.Sprintf("empty value for %s/%s. It is tombstone. Skip decrypt", splitted[0], splitted[1]))
	}

	l.Debug("Restore - Message decrypted", "decrypted", decrypted)

	if len(signature) == 0 {
		return fmt.Errorf("no signature found. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	err := sks.verifySign(signature, decrypted)
	if err != nil {
		return fmt.Errorf("wrong signature. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	l.Debug("Restore - Message verified", "decrypted", decrypted)

	err = self.HandleRestoreMessagesSelfSource(txn, splitted[0], decrypted, []self.RestoreFunc{
		jwtkafka.SelfRestoreMessage,
	})
	if err != nil {
		return err
	}

	return nil
}

func (sks *SelfKafkaSource) Run(store *io.MemoryStore) {
	replicaName := sks.kf.PluginConfig.SelfTopicName
	sks.logger.Debug("Watcher - start", "replica_name", replicaName)
	defer sks.logger.Debug("Watcher - stop", "replica_name", replicaName)

	// groupId := fmt.Sprintf("run-%s", replicaName)
	groupId := replicaName
	rd := sks.kf.GetSubscribedRunConsumer(groupId, replicaName)
	sks.logger.Debug(fmt.Sprintf("Watcher - got consumer %s/%s/%s", groupId, replicaName, replicaName), "replica_name", replicaName)

	sharedkafka.RunMessageLoop(rd, sks.messageHandler(store), sks.stopC, sks.logger)
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
			sks.logger.Debug(fmt.Sprintf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset))
			return
		}

		if len(signature) == 0 {
			sks.logger.Debug(fmt.Sprintf("no signature found. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset))
			return
		}

		err = sks.verifySign(signature, decrypted)
		if err != nil {
			sks.logger.Warn(fmt.Sprintf("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset))
			return
		}

		source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
		if err != nil {
			sks.logger.Debug("build source message failed", err)
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

	sks.logger.Debug(fmt.Sprintf("Handle new message %s/%s", msg.Type, msg.ID), "type", msg.Type, "id", msg.ID)

	err := self.HandleNewMessageSelfSource(tx, sks.handlerFactory(store, tx), msg)
	if err != nil {
		sks.logger.Error(fmt.Sprintf("Error message handle %s/%s: %s", msg.Type, msg.ID, err), "type", msg.Type, "id", msg.ID, "err", err)
		return err
	}

	sks.logger.Debug(fmt.Sprintf("Message handled successful %s/%s", msg.Type, msg.ID), "type", msg.Type, "id", msg.ID)

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
