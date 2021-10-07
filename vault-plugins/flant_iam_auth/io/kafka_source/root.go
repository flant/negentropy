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

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/root"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type RootSourceMsgHandlerFactory func(tx *io.MemoryStoreTxn) root.ModelHandler

type RootKafkaSource struct {
	kf        *sharedkafka.MessageBroker
	decryptor *sharedkafka.Encrypter

	stopC chan struct{}

	logger            hclog.Logger
	msgHandlerFactory RootSourceMsgHandlerFactory
}

func NewRootKafkaSource(kf *sharedkafka.MessageBroker, msgHandlerFactory RootSourceMsgHandlerFactory, logger hclog.Logger) *RootKafkaSource {
	return &RootKafkaSource{
		kf:        kf,
		decryptor: sharedkafka.NewEncrypter(),

		stopC:             make(chan struct{}),
		logger:            logger,
		msgHandlerFactory: msgHandlerFactory,
	}
}

func (rk *RootKafkaSource) Name() string {
	return rk.kf.PluginConfig.RootTopicName
}

func (rk *RootKafkaSource) Restore(txn *memdb.Txn, _ hclog.Logger) error {
	rootTopic := rk.kf.PluginConfig.RootTopicName
	replicaName := rk.kf.PluginConfig.SelfTopicName

	rk.logger.Debug("Restore started", "root topic", rk.kf.PluginConfig.RootTopicName, "self topic", rk.kf.PluginConfig.SelfTopicName)
	defer rk.logger.Debug("Restore finished")

	// groupId := fmt.Sprintf("restore-%s", replicaName)
	groupId := replicaName
	runConsumer := rk.kf.GetConsumer(groupId, rootTopic, false)
	defer runConsumer.Close()

	rk.logger.Debug(fmt.Sprintf("Restore - got consumer %s/%s/%s", groupId, replicaName, rootTopic))

	r := rk.kf.GetRestorationReader(rootTopic)
	defer r.Close()

	rk.logger.Debug("Restore - got restoration reader")

	return sharedkafka.RunRestorationLoop(r, runConsumer, replicaName, txn, rk.restoreMsgHandler, rk.logger)
}

func (rk *RootKafkaSource) restoreMsgHandler(txn *memdb.Txn, msg *kafka.Message, _ hclog.Logger) error {
	l := rk.logger.Named("restoreMsgHandler")
	l.Debug("started")
	defer l.Debug("exit")
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		l.Debug("wrong object key format", "key", msg.Key)
		return fmt.Errorf("key has wong format: %s", msg.Key)
	}

	l.Debug("Restore - messages keys", "keys", splitted)

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

	return root.HandleRestoreMessagesRootSource(txn, splitted[0], decrypted)
}

func (rk *RootKafkaSource) Run(store *io.MemoryStore) {
	rootTopic := rk.kf.PluginConfig.RootTopicName
	replicaName := rk.kf.PluginConfig.SelfTopicName
	rk.logger.Debug("Watcher - start", "root_topic", rootTopic, "replica_name", replicaName)
	defer rk.logger.Debug("Watcher - stop", "root_topic", rootTopic, "replica_name", replicaName)

	// groupId := fmt.Sprintf("run-%s", replicaName)
	groupId := replicaName

	rd := rk.kf.GetConsumer(groupId, rootTopic, false)
	rk.logger.Debug(fmt.Sprintf("Restore - got consumer %s/%s/%s", groupId, replicaName, rootTopic), "root_topic", rootTopic, "replica_name", replicaName)

	sharedkafka.RunMessageLoop(rd, rk.msgHandler(store), rk.stopC, rk.logger)
}

func (rk *RootKafkaSource) msgHandler(store *io.MemoryStore) func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	return func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
		l := rk.logger.Named("msgHandler")
		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			// return fmt.Debugf("key has wrong format: %s", string(msg.Key))
			return
		}
		objType, objId := splitted[0], splitted[1]

		l.Debug("Got message", objType, objId)
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

		var decrypted []byte
		if len(msg.Value) > 0 {
			var err error
			decrypted, err = rk.decryptData(msg.Value, chunked)
			if err != nil {
				l.Error(fmt.Sprintf("can't decrypt message: %v. Skipping: %s in topic: %s at offset %d\n",
					err, msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset))
				return
			}
		} else {
			l.Debug(fmt.Sprintf("empty value for %s/%s. It is tombstone. Skip decrypt", objType, objId))
		}

		if len(signature) == 0 {
			l.Error(fmt.Sprintf("no signature found. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset))
			return
		}

		err := rk.verifySign(signature, decrypted)
		if err != nil {
			l.Error(fmt.Sprintf("wrong signature: %v. Skipping message: %s in topic: %s at offset %d\n",
				err, msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset))
			return
		}

		source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
		if err != nil {
			l.Error(fmt.Sprintf("build source message failed: %s", err))
			return
		}

		operation := func() error {
			msgDecoded := &sharedkafka.MsgDecoded{
				Type: objType,
				ID:   objId,
				Data: decrypted,
			}
			return rk.processMessage(source, store, msgDecoded)
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

func (rk *RootKafkaSource) processMessage(source *sharedkafka.SourceInputMessage, store *io.MemoryStore, msg *sharedkafka.MsgDecoded) error {
	tx := store.Txn(true)
	defer tx.Abort()

	rk.logger.Debug(fmt.Sprintf("Handle new message %s/%s", msg.Type, msg.ID), "type", msg.Type, "id", msg.ID)
	err := root.HandleNewMessageIamRootSource(tx, rk.msgHandlerFactory(tx), msg)
	if err != nil {
		rk.logger.Error(fmt.Sprintf("Error message handle %s/%s: %s", msg.Type, msg.ID, err), "type", msg.Type, "id", msg.ID, "err", err)
		return backoff.Permanent(err)
	}

	rk.logger.Debug(fmt.Sprintf("Message handled successful %s/%s", msg.Type, msg.ID), "type", msg.Type, "id", msg.ID)

	return tx.Commit(source)
}

func (rk *RootKafkaSource) Stop() {
	rk.stopC <- struct{}{}
}
