package io

import (
	"fmt"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	hcmemdb "github.com/hashicorp/go-memdb"

	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type Txn interface {
	Insert(table string, obj interface{}) error
	Delete(table string, obj interface{}) error
	First(table, index string, args ...interface{}) (interface{}, error)
	Get(table, index string, args ...interface{}) (hcmemdb.ResultIterator, error)
}

type KafkaSourceImpl struct {
	// broker connection
	KafkaBroker *sharedkafka.MessageBroker
	// stop chan for stop infinite loop
	StopC chan struct{}
	// Logger
	Logger hclog.Logger
	// is message loop run
	run bool
	// runConsumer GroupID
	ProvideRunConsumerGroupID func(kf *sharedkafka.MessageBroker) string
	// topicName
	ProvideTopicName func(kf *sharedkafka.MessageBroker) string

	// check msg signature
	VerifySign func(signature []byte, messageValue []byte) error
	// verify encrypted message
	VerifyEncrypted bool
	// Decrypt message
	Decrypt func(encryptedMessageValue []byte, chunked bool) ([]byte, error)
	// process MsgDecoded at normal reading loop
	ProcessRunMessage func(txn Txn, m sharedkafka.MsgDecoded) error
	// process MsgDecoded at restoration loop
	ProcessRestoreMessage func(txn Txn, m sharedkafka.MsgDecoded) error

	// is Runnable
	Runnable bool
}

func (rk *KafkaSourceImpl) Name() string {
	return rk.ProvideTopicName(rk.KafkaBroker)
}

func (rk *KafkaSourceImpl) Stop() {
	if rk.run {
		rk.StopC <- struct{}{}
		rk.run = false
	}
}

func (rk *KafkaSourceImpl) Run(store *MemoryStore) {
	if !rk.Runnable || rk.run {
		return
	}

	rk.Logger.Debug("Watcher - start", "groupID", rk.ProvideRunConsumerGroupID(rk.KafkaBroker))
	defer rk.Logger.Debug("Watcher - stop", "groupID", rk.ProvideRunConsumerGroupID(rk.KafkaBroker))
	runConsumer, err := rk.KafkaBroker.GetSubscribedRunConsumer(rk.ProvideRunConsumerGroupID(rk.KafkaBroker), rk.ProvideTopicName(rk.KafkaBroker))
	if err != nil {
		// it is critical error, if it happens, there is no way to restart it without repairing
		rk.Logger.Error(fmt.Sprintf("critical error: %s", err.Error()))
	}

	rk.run = true
	defer sharedkafka.DeferredСlose(runConsumer, rk.Logger)
	rk.runMessageLoop(store, runConsumer)
}

func (rk *KafkaSourceImpl) runMessageLoop(store *MemoryStore, consumer *kafka.Consumer) {
	logger := rk.Logger.Named("RunMessageLoop")
	logger.Info("start")
	defer logger.Info("exit")

	for {
		select {
		case <-rk.StopC:
			logger.Warn("Receive stop signal")
			consumer.Unsubscribe() //nolint:errcheck
			// c.Close()    // closing by DefferedClose()
			return

		case ev := <-consumer.Events():
			switch e := ev.(type) {
			case *kafka.Message:
				rk.msgRunHandler(store, consumer, e)
				// commit is provided through MemStore.Commit(...)

			default:
				logger.Debug(fmt.Sprintf("Recieve not handled event %s", e.String()))
			}
		}
	}
}

func (rk *KafkaSourceImpl) msgRunHandler(store *MemoryStore, sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	decoded, err := rk.decodeMessageAndCheck(msg)
	if err != nil {
		rk.Logger.Error("decoding and checking message:", err)
		return
	}

	rk.Logger.Debug(fmt.Sprintf("got message: %s-%s", decoded.Type, decoded.ID))

	source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
	if err != nil {
		rk.Logger.Error(fmt.Sprintf("build source message failed: %s", err.Error()))
	}

	operation := func() error {
		txn := store.Txn(true)
		defer txn.Abort()
		err := rk.ProcessRunMessage(txn, *decoded)
		if err != nil {
			return err
		}
		return txn.Commit(source)
	}

	err = backoff.Retry(operation, ThirtySecondsBackoff())
	if err != nil {
		rk.Logger.Error(fmt.Sprintf("retries failed:%s", err.Error()))
	}
}

func (rk *KafkaSourceImpl) decodeMessageAndCheck(msg *kafka.Message) (*sharedkafka.MsgDecoded, error) {
	result := &sharedkafka.MsgDecoded{}
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		return nil, fmt.Errorf("key %q has wong format", string(msg.Key))
	}
	result.Type, result.ID = splitted[0], splitted[1]

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

	var err error
	result.Data = msg.Value
	if rk.Decrypt != nil && len(msg.Value) > 0 {
		result.Data, err = rk.Decrypt(msg.Value, chunked)
		if err != nil {
			return nil, fmt.Errorf("decryption: %w", err)
		}
	}

	if rk.VerifySign != nil {
		if len(signature) == 0 {
			return nil, fmt.Errorf("no signature found for: %q in topic %s at offset %d", string(msg.Key), *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
		}
		var toVerify []byte
		if rk.VerifyEncrypted {
			toVerify = msg.Value
		} else {
			toVerify = result.Data
		}
		err := rk.VerifySign(signature, toVerify)
		if err != nil {
			return nil, fmt.Errorf("wrong signature of: %q in topic %s at offset %d", string(msg.Key), *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
		}
	}

	return result, nil
}

func (rk *KafkaSourceImpl) Restore(txn *memdb.Txn) error {
	if rk.run {
		return fmt.Errorf("MultipassGenerationKafkaSource has unstopped main reading loop")
	}

	replicaName := rk.KafkaBroker.PluginConfig.SelfTopicName
	groupID := replicaName
	restorationConsumer, err := rk.KafkaBroker.GetRestorationReader()
	if err != nil {
		return err
	}
	runConsumer, err := rk.KafkaBroker.GetUnsubscribedRunConsumer(groupID)
	if err != nil {
		return err
	}

	defer sharedkafka.DeferredСlose(restorationConsumer, rk.Logger)
	defer sharedkafka.DeferredСlose(runConsumer, rk.Logger)
	return sharedkafka.RunRestorationLoop(restorationConsumer, runConsumer, rk.ProvideTopicName(rk.KafkaBroker),
		txn, rk.msgRestoreHandler, rk.Logger)
}

func (rk *KafkaSourceImpl) msgRestoreHandler(txn *memdb.Txn, msg *kafka.Message, _ hclog.Logger) error {
	decoded, err := rk.decodeMessageAndCheck(msg)
	if err != nil {
		return fmt.Errorf("decoding and checking message: %w", err)
	}

	rk.Logger.Debug(fmt.Sprintf("got message: %s-%s", decoded.Type, decoded.ID))

	operation := func() error {
		return rk.ProcessRestoreMessage(txn, *decoded)
	}

	return backoff.Retry(operation, ThirtySecondsBackoff())
}
