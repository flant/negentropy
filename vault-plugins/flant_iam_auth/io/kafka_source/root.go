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

	logger hclog.Logger
	msgHandlerFactory RootSourceMsgHandlerFactory
}

func NewRootKafkaSource(kf *sharedkafka.MessageBroker, msgHandlerFactory RootSourceMsgHandlerFactory, logger hclog.Logger) *RootKafkaSource {
	return &RootKafkaSource{
		kf:        kf,
		decryptor: sharedkafka.NewEncrypter(),

		stopC: make(chan struct{}),
		logger: logger,
		msgHandlerFactory: msgHandlerFactory,
	}
}

func (rk *RootKafkaSource) Name() string {
	return rk.kf.PluginConfig.RootTopicName
}

func (rk *RootKafkaSource) Restore(txn *memdb.Txn) error {
	rootTopic := rk.kf.PluginConfig.RootTopicName
	replicaName := rk.kf.PluginConfig.SelfTopicName

	rk.logger.Error("Restore started", "root topic", rk.kf.PluginConfig.RootTopicName, "self topic", rk.kf.PluginConfig.SelfTopicName)
	defer rk.logger.Error("Restore finished")

	runConsumer := rk.kf.GetConsumer(replicaName, rootTopic, false)
	defer runConsumer.Close()

	rk.logger.Error("Restore - got consumer")

	r := rk.kf.GetRestorationReader(rootTopic)
	defer r.Close()

	rk.logger.Error("Restore - got restoration reader")

	return sharedkafka.RunRestorationLoop(r, runConsumer, replicaName, txn, rk.restoreMsgHandler)
}

func (rk *RootKafkaSource) restoreMsgHandler(txn *memdb.Txn, msg *kafka.Message) error {
	rk.logger.Error("Restore - handler run")
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		log.Printf("wrong object Key format: %s\n", msg.Key)
		return fmt.Errorf("key has wong format: %s", msg.Key)
	}

	rk.logger.Error("Restore - messages keys", "keys", splitted)

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

	rk.logger.Error("Restore - signature", "sig", signature, "chuncked", chunked)

	decrypted, err := rk.decryptData(msg.Value, chunked)
	if err != nil {
		return fmt.Errorf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	rk.logger.Error("Restore - decrypted msg", "msg", decrypted)

	if len(signature) == 0 {
		return fmt.Errorf("no signature found. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	err = rk.verifySign(signature, decrypted)
	if err != nil {
		return fmt.Errorf("wrong signature. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	rk.logger.Error("Restore - signature verified", decrypted)

	return root.HandleRestoreMessagesRootSource(txn, splitted[0], decrypted)
}

func (rk *RootKafkaSource) Run(store *io.MemoryStore) {

	rootTopic := rk.kf.PluginConfig.RootTopicName
	replicaName := rk.kf.PluginConfig.SelfTopicName
	rk.logger.Error("Watcher - start", "root_topic", rootTopic, "replica_name", replicaName)
	defer func() {
		rk.logger.Error("Watcher - stop", "root_topic", rootTopic, "replica_name", replicaName)
	}()

	rd := rk.kf.GetConsumer(replicaName, rootTopic, false)
	rk.logger.Error("Watcher - got consumer", "root_topic", rootTopic, "replica_name", replicaName)

	sharedkafka.RunMessageLoop(rd, rk.msgHandler(store), rk.stopC, rk.logger)
}

func (rk *RootKafkaSource) msgHandler(store *io.MemoryStore) func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	return func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
		panic("??????????????????????????????????????????????????????????")
		rk.logger.Error("Handle root source message")
		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			// return fmt.Errorf("key has wong format: %s", string(msg.Key))
			return
		}
		objType, objId := splitted[0], splitted[1]

		rk.logger.Error("Message id", objType, objId)
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

		rk.logger.Error("Root message signture and chunked", signature, chunked)

		decrypted, err := rk.decryptData(msg.Value, chunked)
		if err != nil {
			log.Printf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			return
		}

		rk.logger.Error("Root message decrypted data", decrypted)

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

		rk.logger.Error("Root message verified")

		source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
		if err != nil {
			log.Println("build source message failed", err)
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

	rk.logger.Debug("Handle root source message", msg.Type, msg.ID)
	panic("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	err := root.HandleNewMessageIamRootSource(tx, rk.msgHandlerFactory(tx), msg)
	if err != nil {
		rk.logger.Debug("Root source message handle err", msg.Type, msg.ID, err)
		return backoff.Permanent(err)
	}

	rk.logger.Debug("Root source message handled", msg.Type, msg.ID)

	return tx.Commit(source)
}

func (rk *RootKafkaSource) Stop() {
	rk.stopC <- struct{}{}
}
