package io

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type MsgDecoded struct {
	Type string
	ID   string
	Data []byte
}

func (m *MsgDecoded) IsDeleted() bool {
	return len(m.Data) == 0
}

type MessageHandler func(sourceConsumer *kafka.Consumer, msg *kafka.Message)

func RunMessageLoop(c *kafka.Consumer, msgHandler MessageHandler, stopC chan struct{}, logger hclog.Logger) {
	logger = logger.Named("RunMessageLoop")
	logger.Info("start")
	defer logger.Info("exit")

	for {
		select {
		case <-stopC:
			logger.Warn("Receive stop signal")
			c.Unsubscribe() //nolint:errcheck
			// c.Close()    // closing by DefferedClose()
			return

		case ev := <-c.Events():
			switch e := ev.(type) {
			case *kafka.Message:
				msgHandler(c, e)
				// commit is provided through MemStore.Commit(...)

			default:
				logger.Debug(fmt.Sprintf("Recieve not handled event %s", e.String()))
			}
		}
	}
}

// RunRestorationLoop read from topic untill runConsumer and handle with handler each message.
// runConsumer after using at RunRestorationLoop can be used, but need Subscribe(topic)
func RunRestorationLoop(newConsumer, runConsumer *kafka.Consumer, topicName string, txn *memdb.Txn,
	handler func(txn *memdb.Txn, msg *kafka.Message, logger hclog.Logger) error, logger hclog.Logger) error {
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

func RunRestorationLoopWITH_LOGS(newConsumer, runConsumer *kafka.Consumer, topicName string, txn *memdb.Txn,
	handler func(txn *memdb.Txn, msg *kafka.Message, logger hclog.Logger) error, logger hclog.Logger) error {
	logger = logger.Named("RunRestorationLoopWITH_LOGS")
	logger.Debug("started", "topicName", topicName)
	defer logger.Debug("exit")

	var lastOffset, edgeOffset int64
	var partition int32
	var err error
	if runConsumer != nil {
		logger.Debug("TODO REMOVE: runConsumer != nil")
		lastOffset, edgeOffset, partition, err = LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topicName)
		if err != nil {
			logger.Debug("TODO REMOVE: getting offset by RunConsumer: Error:%s", err.Error())
			return fmt.Errorf("getting offset by RunConsumer:%w", err)
		}
		logger.Debug("TODO REMOVE: LastAndEdgeOffsetsByRunConsumer", "lastOffset", lastOffset, "edgeOffset", edgeOffset)
	} else {
		logger.Debug("TODO REMOVE: runConsumer == nil")
		lastOffset, partition, err = LastOffsetByNewConsumer(newConsumer, topicName)
		if err != nil {
			logger.Debug("TODO REMOVE: getting offset by newConsumer: Error:%s", err.Error())
			return fmt.Errorf("getting offset by newConsumer:%w", err)
		}
		logger.Debug("TODO REMOVE: LastOffsetByNewConsumer", "lastOffset", lastOffset, "edgeOffset", edgeOffset)
	}
	logger.Debug("TODO REMOVE", "lastOffset", lastOffset, "edgeOffset", edgeOffset)

	if lastOffset == 0 && edgeOffset == 0 {
		logger.Debug("normal finish: no messages", "topicName", topicName)
		return nil
	}
	newConsumer.Unassign() // nolint:errcheck
	err = setNewConsumerToBeginning(newConsumer, topicName, partition)
	logger.Debug("setNewConsumerToBeginning: err = %v", err)
	if err != nil {
		return err
	}

	c := newConsumer.Events()
	consumed := 0
	logger.Debug("TODO REMOVE", "lastOffset", lastOffset, "edgeOffset", edgeOffset)
	logger.Debug("TODO REMOVE", "partition", partition)
	for {
		logger.Debug("TODO REMOVE, Enter loop")
		var msg *kafka.Message
		ev := <-c
		logger.Debug(fmt.Sprintf("got event:%s", ev.String()))
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
		logger.Debug("end loop", "currentMessageOffset", currentMessageOffset)
	}
}

// returns metadata & last offset in topic
// Note works only for 1 partition
func getNextWritingOffsetByMetaData(consumer *kafka.Consumer, topicName string) (meta *kafka.Metadata, edgeOffset int64, err error) {
	meta, err = getMetaDataWithRetry(consumer, topicName)
	if err != nil {
		return
	}

	topicMeta := meta.Topics[topicName]
	if topicMeta.Topic == "" || len(topicMeta.Partitions) == 0 {
		return nil, 0, fmt.Errorf("getMeta returns empty response, probably topic %s is not exists", topicName)
	}
	if len(topicMeta.Partitions) != 1 {
		return nil, 0, fmt.Errorf("topic %s has %d partiotions, the program allows only 1 partition", topicName,
			len(topicMeta.Partitions))
	}
	for _, partition := range topicMeta.Partitions {
		lastPartitionOffset, err := queryWatermarkOffsetsWithRetry(consumer, topicName, partition)
		if err != nil {
			return nil, 0, err
		}
		if lastPartitionOffset > edgeOffset {
			edgeOffset = lastPartitionOffset
		}
	}
	return meta, edgeOffset, nil
}

func queryWatermarkOffsetsWithRetry(consumer *kafka.Consumer, topicName string, partition kafka.PartitionMetadata) (int64, error) {
	var lastPartitionOffset int64
	err := backoff.Retry(func() error {
		var err error
		_, lastPartitionOffset, err = consumer.QueryWatermarkOffsets(topicName, partition.ID, 500)
		return err
	}, thirtySecondsBackoff())
	if err != nil {
		return 0, fmt.Errorf("query watermark offsets for topic: %q at partition %q: %w", topicName, partition.ID, err)
	}
	return lastPartitionOffset, err
}

func getMetaDataWithRetry(consumer *kafka.Consumer, topicName string) (*kafka.Metadata, error) {
	var meta *kafka.Metadata
	err := backoff.Retry(func() error {
		var err error
		meta, err = consumer.GetMetadata(&topicName, false, 500)
		return err
	}, thirtySecondsBackoff())
	if err != nil {
		return nil, fmt.Errorf("getting metadata for topic: %q: %w", topicName, err)
	}
	return meta, nil
}

// LastAndEdgeOffsetsByRunConsumer
// note: works only for 1 partition
func LastAndEdgeOffsetsByRunConsumer(runConsumer *kafka.Consumer, newConsumer *kafka.Consumer,
	topicName string) (lastOffsetForRestoring int64, edgeOffset int64, partition int32, err error) {
	metaData, t, err := getNextWritingOffsetByMetaData(runConsumer, topicName)
	if err != nil {
		return 0, 0, 0, err
	}
	partition = metaData.Topics[topicName].Partitions[0].ID
	if t == 0 {
		return 0, 0, partition, nil
	}
	tp := kafka.TopicPartition{
		Topic:     &topicName,
		Partition: partition,
	}

	offsets, err := runConsumer.Committed([]kafka.TopicPartition{tp}, 10000)
	if err != nil {
		return 0, 0, 0, err
	}
	nextOffset := offsets[0].Offset
	if nextOffset < 0 {
		return 0, 0, partition, nil
	}

	tp = kafka.TopicPartition{
		Topic:     &topicName,
		Partition: metaData.Topics[topicName].Partitions[0].ID,
		Offset:    kafka.OffsetTail(2),
	}
	err = newConsumer.Assign([]kafka.TopicPartition{tp})
	defer newConsumer.Unassign() // nolint:errcheck
	if err != nil {
		return 0, 0, 0, fmt.Errorf("assigning last message: %w", err)
	}
	msg, err := getMessageWitRetry(newConsumer)
	if err != nil {
		return 0, 0, 0, err
	}

	lastOffsetAtTopic := msg.TopicPartition.Offset
	if nextOffset > lastOffsetAtTopic {
		return int64(lastOffsetAtTopic), 0, partition, nil
	}
	return 0, int64(nextOffset), partition, nil
}

func getMessageWitRetry(newConsumer *kafka.Consumer) (*kafka.Message, error) {
	var msg *kafka.Message
	err := backoff.Retry(func() error {
		var err error
		msg, err = getMessage(newConsumer)
		return err
	}, thirtySecondsBackoff())
	if err != nil {
		return nil, fmt.Errorf("getting message: %w", err)
	}
	return msg, nil
}

func getMessage(newConsumer *kafka.Consumer) (*kafka.Message, error) {
	ch := newConsumer.Events()
	timerCh := time.NewTimer(time.Millisecond * 100).C
	var msg *kafka.Message
	for msg == nil {
		fmt.Print("\n====== ENTER ENDLESS LOOP  ======\n\n")
		select {
		case ev := <-ch:
			switch e := ev.(type) {
			case *kafka.Message:
				msg = e
			default:
				return nil, fmt.Errorf("recieve not handled event %s", e.String())
			}
		case <-timerCh:
			return nil, fmt.Errorf("out by timer 100ms")
		}
	}
	return msg, nil
}

func setNewConsumerToBeginning(consumer *kafka.Consumer, topicName string, partition int32) error {
	tp := kafka.TopicPartition{
		Topic:     &topicName,
		Partition: partition,
		Offset:    kafka.OffsetBeginning,
	}
	err := consumer.Assign([]kafka.TopicPartition{tp})
	if err != nil {
		return fmt.Errorf("assigning first message: %w", err)
	}
	return nil
}

func thirtySecondsBackoff() backoff.BackOff {
	backoffRequest := backoff.NewExponentialBackOff()
	backoffRequest.MaxElapsedTime = time.Second * 30
	return backoffRequest
}
