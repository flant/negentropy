package kafka

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const topicName = "jwks"

type JWKSKafkaSource struct {
	kf *sharedkafka.MessageBroker

	stopC chan struct{}

	logger hclog.Logger
}

func NewJWKSKafkaSource(kf *sharedkafka.MessageBroker, logger hclog.Logger) *JWKSKafkaSource {
	return &JWKSKafkaSource{
		kf: kf,

		stopC:  make(chan struct{}),
		logger: logger,
	}
}

func (rk *JWKSKafkaSource) Name() string {
	return topicName
}

func (rk *JWKSKafkaSource) Restore(txn *memdb.Txn) error {
	runConsumer := rk.kf.GetConsumer(rk.kf.PluginConfig.SelfTopicName, topicName, false)
	defer runConsumer.Close()

	r := rk.kf.GetRestorationReader(topicName)
	defer r.Close()

	rk.logger.Debug("Restore - got restoration reader")

	return sharedkafka.RunRestorationLoop(r, runConsumer, topicName, txn, rk.restoreMsgHandler)
}

func (rk *JWKSKafkaSource) restoreMsgHandler(txn *memdb.Txn, msg *kafka.Message) error {
	rk.logger.Debug("Restore - handler run")
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		rk.logger.Error("wrong object Key format: %s\n", msg.Key)
		return fmt.Errorf("key has wong format: %s", msg.Key)
	}

	rk.logger.Debug("Restore - messages keys", "keys", splitted)
	objType := splitted[0]
	objId := splitted[1]

	if len(msg.Value) == 0 {
		return nil
	}

	if objType != jwt.JWKSType {
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

	var jwks *jwt.JWKS

	err = json.Unmarshal(msg.Value, jwks)
	if err != nil {
		return err
	}

	return txn.Insert(objType, objId)
}

func (rk *JWKSKafkaSource) Run(store *io.MemoryStore) {
	rd := rk.kf.GetConsumer(rk.kf.PluginConfig.SelfTopicName, topicName, false)

	sharedkafka.RunMessageLoop(rd, rk.msgHandler(store), rk.stopC, rk.logger)
}

func (rk *JWKSKafkaSource) msgHandler(store *io.MemoryStore) func(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
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
			rk.logger.Warn("no signature found. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			return
		}

		err := rk.verifySign(signature, msg.Value)
		if err != nil {
			rk.logger.Warn("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
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
		err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], signature)
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

	if msg.IsDeleted() {
		obj, err := tx.First(jwt.JWKSType, "id", msg.ID)
		if err != nil {
			return err
		}
		err = tx.Delete(jwt.JWKSType, obj)
		if err != nil {
			return err
		}

		return tx.Commit(source)
	}

	// creation

	var jwks *jwt.JWKS

	err := json.Unmarshal(msg.Data, jwks)
	if err != nil {
		return backoff.Permanent(err)
	}

	err = tx.Insert(jwt.JWKSType, jwks)
	if err != nil {
		return err
	}

	return tx.Commit(source)
}

func (rk *JWKSKafkaSource) Stop() {
	rk.stopC <- struct{}{}
}
