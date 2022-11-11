package io

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	hcmemdb "github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/logical"

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
	// name - should be unique in plugin, mandatory
	NameOfSource string
	// broker connection, mandatory
	KafkaBroker *sharedkafka.MessageBroker
	// Logger, mandatory
	Logger hclog.Logger
	// runConsumer GroupID, mandatory
	ProvideRunConsumerGroupID func(kf *sharedkafka.MessageBroker) string
	// topicName, mandatory
	ProvideTopicName func(kf *sharedkafka.MessageBroker) string
	// check msg signature, mandatory
	VerifySign func(signature []byte, messageValue []byte) error
	// Decrypt message
	Decrypt func(encryptedMessageValue []byte, chunked bool) ([]byte, error)
	// process MsgDecoded at normal reading loop, if RestoreStrictlyTillRunConsumer=true,
	// should be idempotent to situation when ProcessRestoreMessage works first on message
	ProcessRunMessage func(txn Txn, m MsgDecoded) error
	// process MsgDecoded at restoration loop, mandatory
	ProcessRestoreMessage func(txn Txn, m MsgDecoded) error

	// what to set at SourceInputMessage at field IgnoreBody at usual message loop
	IgnoreSourceInputMessageBody bool

	// is Runnable
	Runnable bool
	// stop chan for stop infinite loop, created by Run method
	stopC chan struct{}
	// is message loop run, operate by methods
	run bool
	// if true, Restoration loop skip processing message, Run loop on verify error just provide message
	SkipRestorationOnWrongSignature bool

	// set true for saving processed by run consumer offset and
	RestoreStrictlyTillRunConsumer bool
	// can be nil if not RestoreStrictlyTillRunConsumer
	Storage logical.Storage
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
				err := rk.msgRunHandler(store, consumer, e)
				// commit is provided through MemStore.Commit(...)
				if errors.Is(err, errWrongSignature) && rk.SkipRestorationOnWrongSignature {
					rk.Logger.Debug(fmt.Sprintf("%s: message skiped", err.Error()))
					err = nil
				}
				if err != nil {
					rk.Logger.Error(fmt.Sprintf("msg: %s: %s", string(e.Key), err.Error()))
				}
				if rk.RestoreStrictlyTillRunConsumer {
					err = StoreLastOffsetToStorage(context.Background(),
						rk.Storage, rk.ProvideRunConsumerGroupID(rk.KafkaBroker), rk.ProvideTopicName(rk.KafkaBroker),
						int64(e.TopicPartition.Offset))
					if err != nil {
						rk.Logger.Error(fmt.Sprintf("storing offset (%d) for: %s: %s", e.TopicPartition.Offset,
							string(e.Key), err.Error()))
					}
				}
			default:
				logger.Debug(fmt.Sprintf("Recieve not handled event %s", e.String()))
			}
		}
	}
}

func (rk *KafkaSourceImpl) msgRunHandler(store *MemoryStore, sourceConsumer *kafka.Consumer, msg *kafka.Message) error {
	decoded, err := rk.decodeMessageAndCheck(msg)
	if err != nil {
		return fmt.Errorf("decoding and checking message: %w", err)
	}

	rk.Logger.Debug(fmt.Sprintf("got message: %s/%s", decoded.Type, decoded.ID))

	source, err := sharedkafka.NewSourceInputMessage(sourceConsumer, msg.TopicPartition)
	if err != nil {
		return fmt.Errorf("build source message failed: %w", err)
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
		return fmt.Errorf("retries failed: %w", err)
	}
	return nil
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
		return fmt.Errorf("%s has unstopped main reading loop", rk.Name())
	}

	restorationConsumer, err := rk.KafkaBroker.GetRestorationReader()
	if err != nil {
		return err
	}
	defer sharedkafka.DeferredСlose(restorationConsumer, rk.Logger)

	return rk.RunRestorationLoop(restorationConsumer, txn, rk.msgRestoreHandler, rk.Logger)
}

// RunRestorationLoop read from topic untill runConsumer or untill the end of topic
func (rk *KafkaSourceImpl) RunRestorationLoop(newConsumer *kafka.Consumer, txn Txn,
	handler func(txn Txn, msg *kafka.Message, logger hclog.Logger) error, logger hclog.Logger) error {
	logger = logger.Named("RunRestorationLoop")
	topicName := rk.ProvideTopicName(rk.KafkaBroker)
	logger.Debug("started", "topicName", topicName)
	defer logger.Debug("exit")
	runConsumerID := rk.ProvideRunConsumerGroupID(rk.KafkaBroker)

	var lastProcessedOffset int64
	var partition int32
	var err error
	if rk.RestoreStrictlyTillRunConsumer {
		lastProcessedOffset, err = LastOffsetFromStorage(context.Background(), rk.Storage, runConsumerID, topicName)
		if err != nil {
			return fmt.Errorf("getting last offset from storage:%w", err)
		}
		logger.Debug("LastOffsetFromStorage", "topicName", topicName, "runConsumerID", runConsumerID, "lastProcessedOffset", lastProcessedOffset) // TODO REMOVE
	} else {
		lastProcessedOffset, partition, err = LastOffsetByNewConsumer(newConsumer, topicName)
		if err != nil {
			return fmt.Errorf("getting offset by newConsumer:%w", err)
		}

	}

	if lastProcessedOffset <= 0 {
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
		err = handler(txn, msg, logger)
		consumed++
		if err != nil {
			return err
		}
		if currentMessageOffset == lastProcessedOffset {
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
