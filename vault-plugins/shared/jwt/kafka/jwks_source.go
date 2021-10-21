package kafka

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const topicName = "jwks"

type JWKSKafkaSource struct {
	kf *sharedkafka.MessageBroker

	stopC chan struct{}

	logger hclog.Logger
	run    bool
}

func NewJWKSKafkaSource(kf *sharedkafka.MessageBroker, parentLogger hclog.Logger) *JWKSKafkaSource {
	return &JWKSKafkaSource{
		kf: kf,

		stopC:  make(chan struct{}),
		logger: parentLogger.Named("jwksKafkaSource"),
	}
}

func (rk *JWKSKafkaSource) Name() string {
	return topicName
}

func (rk *JWKSKafkaSource) Restore(txn *memdb.Txn) error {
	if rk.run {
		return fmt.Errorf("JWKSKafkaSource has unstopped main reading loop")
	}

	replicaName := rk.kf.PluginConfig.SelfTopicName
	groupID := replicaName

	restorationConsumer := rk.kf.GetRestorationReader()
	runConsumer := rk.kf.GetUnsubscribedRunConsumer(groupID)

	defer sharedkafka.DeferredСlose(restorationConsumer, rk.logger)
	defer sharedkafka.DeferredСlose(runConsumer, rk.logger)
	return sharedkafka.RunRestorationLoop(restorationConsumer, runConsumer, topicName, txn, rk.restoreMsgHandler, rk.logger)
}

func (rk *JWKSKafkaSource) restoreMsgHandler(txn *memdb.Txn, msg *kafka.Message, _ hclog.Logger) error {
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

	if objType != model.JWKSType {
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

	jwks := &model.JWKS{}

	err = json.Unmarshal(msg.Value, jwks)
	if err != nil {
		return err
	}

	return txn.Insert(model.JWKSType, jwks)
}

func (rk *JWKSKafkaSource) Run(store *io.MemoryStore) {
	if rk.run {
		return
	}

	replicaName := rk.kf.PluginConfig.SelfTopicName
	rk.logger.Debug("Watcher - start", "replica_name", replicaName)
	defer rk.logger.Debug("Watcher - stop", "replica_name", replicaName)
	groupID := replicaName
	runConsumer := rk.kf.GetSubscribedRunConsumer(groupID, topicName)

	rk.run = true
	defer sharedkafka.DeferredСlose(runConsumer, rk.logger)
	sharedkafka.RunMessageLoop(runConsumer, rk.messageHandler(store), rk.stopC, rk.logger)
}

func (rk *JWKSKafkaSource) messageHandler(store *io.MemoryStore) func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	return func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			// return fmt.Debugf("key has wong format: %s", string(msg.Key))
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

func (rk *JWKSKafkaSource) verifySign(signature []byte, data []byte) error {
	hashed := sha256.Sum256(data)

	for _, pub := range rk.kf.PluginConfig.PeersPublicKeys {
		err := sharedkafka.VerifySignature(signature, pub, hashed)
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("no public key for signature found")
}

func (rk *JWKSKafkaSource) processMessage(source *sharedkafka.SourceInputMessage, store *io.MemoryStore, msg *sharedkafka.MsgDecoded) error {
	tx := store.Txn(true)
	defer tx.Abort()

	rk.logger.Debug(fmt.Sprintf("Handle new message %s/%s", msg.Type, msg.ID), "type", msg.Type, "id", msg.ID)

	source.IgnoreBody = true

	if msg.IsDeleted() {
		obj, err := tx.First(model.JWKSType, "id", msg.ID)
		if err != nil {
			return err
		}
		if obj == nil {
			return nil
		}
		err = tx.Delete(model.JWKSType, obj)
		if err != nil {
			return err
		}

		return tx.Commit(source)
	}

	// creation

	jwks := &model.JWKS{}

	err := json.Unmarshal(msg.Data, jwks)
	if err != nil {
		return backoff.Permanent(err)
	}

	err = tx.Insert(model.JWKSType, jwks)
	if err != nil {
		return err
	}

	return tx.Commit(source)
}

func (rk *JWKSKafkaSource) Stop() {
	if rk.run {
		rk.stopC <- struct{}{}
		rk.run = false
	}
}
