package io

import (
	"errors"
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
	// name - should be unique in plugin
	NameOfSource string
	// broker connection
	KafkaBroker *sharedkafka.MessageBroker
	// Logger
	Logger hclog.Logger
	// runConsumer GroupID
	ProvideRunConsumerGroupID func(kf *sharedkafka.MessageBroker) string
	// topicName
	ProvideTopicName func(kf *sharedkafka.MessageBroker) string
	// check msg signature
	VerifySign func(signature []byte, messageValue []byte) error
	// Decrypt message
	Decrypt func(encryptedMessageValue []byte, chunked bool) ([]byte, error)
	// process MsgDecoded at normal reading loop
	ProcessRunMessage func(txn Txn, m MsgDecoded) error
	// process MsgDecoded at restoration loop
	ProcessRestoreMessage func(txn Txn, m MsgDecoded) error

	// what to set at SourceInputMessage at field IgnoreBody at usual message loop
	IgnoreSourceInputMessageBody bool

	// is Runnable
	Runnable bool
	// stop chan for stop infinite loop, created by Run method
	stopC chan struct{}
	// is message loop run
	run bool
	// if true, Restoration loop skip processing message, Run loop on verify error just provide message
	SkipRestorationOnWrongSignature bool
}

func (rk *KafkaSourceImpl) Name() string {
	return rk.NameOfSource
}

func (rk *KafkaSourceImpl) Stop() {
	if rk.run {
		rk.stopC <- struct{}{}
		rk.run = false
	}
}

func (rk *KafkaSourceImpl) Run(store *MemoryStore) {
	if !rk.Runnable || rk.run {
		return
	}
	rk.stopC = make(chan struct{})

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
		case <-rk.stopC:
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
	if errors.Is(err, errWrongSignature) && rk.SkipRestorationOnWrongSignature {
		rk.Logger.Debug(fmt.Sprintf("decoding and checking message: %s: message skiped", err.Error()))
	}
	if err != nil {
		rk.Logger.Error(fmt.Sprintf("decoding and checking message: %s", err.Error()))
		return
	}

	rk.Logger.Debug(fmt.Sprintf("got message: %s/%s", decoded.Type, decoded.ID))

	source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
	if err != nil {
		rk.Logger.Error(fmt.Sprintf("build source message failed: %s", err.Error()))
		return
	}
	source.IgnoreBody = rk.IgnoreSourceInputMessageBody

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

var errWrongSignature = fmt.Errorf("wrong signature")

func (rk *KafkaSourceImpl) decodeMessageAndCheck(msg *kafka.Message) (*MsgDecoded, error) {
	result := &MsgDecoded{}
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
		err := rk.VerifySign(signature, result.Data)
		if err != nil {
			return nil, fmt.Errorf("%w: %q in topic %s at offset %d", errWrongSignature, string(msg.Key), *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
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
	return rk.RunRestorationLoop(restorationConsumer, runConsumer, rk.ProvideTopicName(rk.KafkaBroker),
		txn, rk.msgRestoreHandler, rk.Logger)
}

// RunRestorationLoop read from topic untill runConsumer and handle with handler each message.
// runConsumer after using at RunRestorationLoop can be used, but need Subscribe(topic)
func (rk *KafkaSourceImpl) RunRestorationLoop(newConsumer, runConsumer *kafka.Consumer, topicName string, txn Txn,
	handler func(txn Txn, msg *kafka.Message, logger hclog.Logger) error, logger hclog.Logger) error {
	logger = logger.Named("RunRestorationLoop")
	logger.Debug("started", "topicName", topicName)
	defer logger.Debug("exit")

	var lastOffset, edgeOffset int64
	var partition int32
	var err error
	if runConsumer != nil {
		lastOffset, edgeOffset, partition, err = LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topicName)
		if err != nil {
			return fmt.Errorf("getting offset by RunConsumer:%w", err)
		}
	} else {
		lastOffset, partition, err = LastOffsetByNewConsumer(newConsumer, topicName)
		if err != nil {
			return fmt.Errorf("getting offset by newConsumer:%w", err)
		}
	}

	if lastOffset == 0 && edgeOffset == 0 {
		logger.Debug("normal finish: no messages", "topicName", topicName)
		return nil
	}
	newConsumer.Unassign() // nolint:errcheck
	err = setNewConsumerToBeginning(newConsumer, topicName, partition)
	if err != nil {
		return err
	}

	c := newConsumer.Events()
	consumed := 0
	for {
		var msg *kafka.Message
		ev := <-c

		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
		default:
			logger.Debug(fmt.Sprintf("Recieve not handled event %s", e.String()))
			continue
		}
		currentMessageOffset := int64(msg.TopicPartition.Offset)
		if edgeOffset > 0 && currentMessageOffset >= edgeOffset {
			logger.Info(fmt.Sprintf("topicName: %s - normal finish, consumed %d messages", topicName, consumed))
			return nil
		}
		err := handler(txn, msg, logger)
		consumed++
		if err != nil {
			return err
		}
		if lastOffset > 0 && currentMessageOffset == lastOffset {
			logger.Info(fmt.Sprintf("topicName: %s - normal finish, consumed %d", topicName, consumed))
			return nil
		}
	}
}

func (rk *KafkaSourceImpl) msgRestoreHandler(txn Txn, msg *kafka.Message, _ hclog.Logger) error {
	decoded, err := rk.decodeMessageAndCheck(msg)
	if errors.Is(err, errWrongSignature) && rk.SkipRestorationOnWrongSignature {
		rk.Logger.Debug(fmt.Sprintf("decoding and checking message: %s: message skiped", err.Error()))
		return nil
	}
	if err != nil {
		return fmt.Errorf("decoding and checking message: %w", err)
	}

	rk.Logger.Debug(fmt.Sprintf("got message: %s/%s", decoded.Type, decoded.ID))

	operation := func() error {
		return rk.ProcessRestoreMessage(txn, *decoded)
	}

	return backoff.Retry(operation, ThirtySecondsBackoff())
}
