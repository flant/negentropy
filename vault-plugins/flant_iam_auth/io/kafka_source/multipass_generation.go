package kafka_source

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type MultipassGenerationKafkaSource struct {
	kf *sharedkafka.MessageBroker

	stopC chan struct{}

	logger hclog.Logger
	run    bool
}

func NewMultipassGenerationSource(kf *sharedkafka.MessageBroker, logger hclog.Logger) *MultipassGenerationKafkaSource {
	return &MultipassGenerationKafkaSource{
		kf: kf,

		stopC:  make(chan struct{}),
		logger: logger,
	}
}

func (rk *MultipassGenerationKafkaSource) Name() string {
	return io.MultipassNumberGenerationTopic
}

func (rk *MultipassGenerationKafkaSource) Restore(txn *memdb.Txn) error {
	runConsumer := rk.kf.GetUnsubscribedRunConsumer(rk.kf.PluginConfig.SelfTopicName)
	defer runConsumer.Close()

	restorationConsumer := rk.kf.GetRestorationReader()
	defer restorationConsumer.Close()

	return sharedkafka.RunRestorationLoop(restorationConsumer, runConsumer, io.MultipassNumberGenerationTopic, txn, rk.restoreMsgHandler, rk.logger)
}

func (rk *MultipassGenerationKafkaSource) restoreMsgHandler(txn *memdb.Txn, msg *kafka.Message, _ hclog.Logger) error {
	rk.logger.Debug("Restore - handler run")
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		rk.logger.Error("wrong object Key format: %s\n", msg.Key)
		return fmt.Errorf("key has wong format: %s", msg.Key)
	}

	rk.logger.Debug("Restore - messages keys", "keys", splitted)
	objType := splitted[0]

	if len(msg.Value) == 0 {
		return nil
	}

	if objType != model.MultipassGenerationNumberType {
		return nil
	}

	var signature []byte
	for _, header := range msg.Headers {
		if header.Key == "signature" {
			signature = header.Value
		}
	}

	if len(signature) == 0 {
		return fmt.Errorf("no signature found. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	err := rk.verifySign(signature, msg.Value)
	if err != nil {
		return fmt.Errorf("wrong signature. Skipping message: %s in topic: %s at offset %d\n", msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
	}

	multipassGenerationNumber := &model.MultipassGenerationNumber{}

	err = json.Unmarshal(msg.Value, multipassGenerationNumber)
	if err != nil {
		return err
	}

	return txn.Insert(model.MultipassGenerationNumberType, multipassGenerationNumber)
}

func (rk *MultipassGenerationKafkaSource) Run(store *sharedio.MemoryStore) {
	rd := rk.kf.GetSubscribedRunConsumer(rk.kf.PluginConfig.SelfTopicName, io.MultipassNumberGenerationTopic)

	rk.run = true
	sharedkafka.RunMessageLoop(rd, rk.msgHandler(store), rk.stopC, rk.logger)
}

func (rk *MultipassGenerationKafkaSource) msgHandler(store *sharedio.MemoryStore) func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	return func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			rk.logger.Debug("key has wong format", string(msg.Key))
			return
		}
		objType, objId := splitted[0], splitted[1]

		rk.logger.Debug("Got message", objType, objId)
		var signature []byte
		for _, header := range msg.Headers {
			if header.Key == "signature" {
				signature = header.Value
			}
		}

		if len(signature) == 0 {
			rk.logger.Warn(fmt.Sprintf("no signature found. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset))
			return
		}

		err := rk.verifySign(signature, msg.Value)
		if err != nil {
			rk.logger.Warn(fmt.Sprintf("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset))
			return
		}

		source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
		if err != nil {
			rk.logger.Error("build source message failed", err)
			return
		}
		msgDecoded := &sharedkafka.MsgDecoded{
			Type: objType,
			ID:   objId,
			Data: msg.Value,
		}

		operation := func() error {
			return rk.processMessage(source, store, msgDecoded)
		}
		err = backoff.Retry(operation, backoff.NewExponentialBackOff())
		if err != nil {
			panic(err)
		}
	}
}

func (rk *MultipassGenerationKafkaSource) verifySign(signature []byte, data []byte) error {
	hashed := sha256.Sum256(data)

	for _, pub := range rk.kf.PluginConfig.PeersPublicKeys {
		err := sharedkafka.VerifySignature(signature, pub, hashed)
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("no public key for signature found")
}

func (rk *MultipassGenerationKafkaSource) processMessage(source *sharedkafka.SourceInputMessage, store *sharedio.MemoryStore, msg *sharedkafka.MsgDecoded) error {
	tx := store.Txn(true)
	defer tx.Abort()

	rk.logger.Debug(fmt.Sprintf("Handle new message %s/%s", msg.Type, msg.ID), "type", msg.Type, "id", msg.ID)

	source.IgnoreBody = true

	if msg.IsDeleted() {
		obj, err := tx.First(model.MultipassGenerationNumberType, repo.ID, msg.ID)
		if err != nil {
			return err
		}
		err = tx.Delete(model.MultipassGenerationNumberType, obj)
		if err != nil {
			return err
		}

		return tx.Commit(source)
	}

	// creation

	jwks := &model.MultipassGenerationNumber{}

	err := json.Unmarshal(msg.Data, jwks)
	if err != nil {
		return backoff.Permanent(err)
	}

	err = tx.Insert(model.MultipassGenerationNumberType, jwks)
	if err != nil {
		return err
	}

	return tx.Commit(source)
}

func (rk *MultipassGenerationKafkaSource) Stop() {
	if rk.run {
		rk.stopC <- struct{}{}
		rk.run = false
	}
}
