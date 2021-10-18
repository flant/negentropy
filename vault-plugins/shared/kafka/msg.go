package kafka

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"
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

// returns metadata & last offset in topic
func getNextWritingOffsetByMetaData(consumer *kafka.Consumer, topicName string) (*kafka.Metadata, int64, error) {
	edgeOffset := int64(0)
	meta, err := consumer.GetMetadata(&topicName, false, -1)
	if err != nil {
		return nil, 0, err
	}

	topicMeta := meta.Topics[topicName]
	for _, partition := range topicMeta.Partitions {
		_, lastPartitionOffset, err := consumer.QueryWatermarkOffsets(topicName, partition.ID, -1)
		if err != nil {
			return nil, 0, err
		}
		if lastPartitionOffset > edgeOffset {
			edgeOffset = lastPartitionOffset
		}
	}
	return meta, edgeOffset, nil
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
	ch := newConsumer.Events()
	var msg *kafka.Message
	for msg == nil {
		ev := <-ch

		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
		default:
			return 0, 0, 0, fmt.Errorf("recieve not handled event %s", e.String())
		}
	}

	lastOffsetAtTopic := msg.TopicPartition.Offset
	if nextOffset > lastOffsetAtTopic {
		return int64(lastOffsetAtTopic), 0, partition, nil
	}
	return 0, int64(nextOffset), partition, nil
}

func LastOffsetByNewConsumer(consumer *kafka.Consumer, topicName string) (lastOffsetForRestoring int64, partition int32, err error) {
	metaData, t, err := getNextWritingOffsetByMetaData(consumer, topicName)
	if err != nil {
		return 0, 0, err
	}
	partition = metaData.Topics[topicName].Partitions[0].ID
	if t == 0 {
		return 0, partition, nil
	}

	tp := kafka.TopicPartition{
		Topic:     &topicName,
		Partition: partition,
		Offset:    kafka.OffsetTail(2),
	}
	err = consumer.Assign([]kafka.TopicPartition{tp})
	if err != nil {
		return 0, 0, fmt.Errorf("assigning last message: %w", err)
	}
	ch := consumer.Events()
	var msg *kafka.Message
	ev := <-ch
	switch e := ev.(type) {
	case *kafka.Message:
		msg = e
	default:
		// do nothing
	}
	lastOffsetAtTopic := msg.TopicPartition.Offset
	return int64(lastOffsetAtTopic), partition, nil
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
