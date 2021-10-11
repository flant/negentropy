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
			c.Close()       //nolint:errcheck
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
// runConsumer is used only for getting edgeOffset by uncommited read, but it should not used after passing
// to RunRestorationLoop
func RunRestorationLoop(newConsumer, runConsumer *kafka.Consumer, topicName string, txn *memdb.Txn,
	handler func(txn *memdb.Txn, msg *kafka.Message, logger hclog.Logger) error, logger hclog.Logger) error {
	logger = logger.Named("RunRestorationLoop")
	logger.Debug("started", "topicName", topicName)
	defer logger.Debug("exit")

	var lastOffset, edgeOffset int64
	var err error
	if runConsumer != nil {
		lastOffset, edgeOffset, err = LastAndEdgeOffsetsByRunConsumer(runConsumer, topicName)
		if err != nil {
			return fmt.Errorf("getting offset by RunConsumer:%w", err)
		}
		err = newConsumer.Subscribe(topicName, nil)
		if err != nil {
			return fmt.Errorf("subscribing newConsumer:%w", err)
		}
	} else {
		lastOffset, err = LastOffsetByNewConsumer(newConsumer, topicName)
		if err != nil {
			return fmt.Errorf("getting offset by newConsumer:%w", err)
		}
		err = resetConsumerToBeginning(newConsumer, topicName)
		if err != nil {
			return err
		}
	}

	if lastOffset == 0 && edgeOffset == 0 {
		logger.Debug("normal finish: no messages", "topicName", topicName)
		return nil
	}

	c := newConsumer.Events()
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
			logger.Debug("normal finish", "topicName", topicName)
			return nil
		}
		err := handler(txn, msg, logger)
		if err != nil {
			return err
		}
		if lastOffset > 0 && currentMessageOffset == lastOffset {
			logger.Debug("normal finish", "topicName", topicName)
			return nil
		}
	}
}

func getNextWritingOffsetByMetaData(consumer *kafka.Consumer, topicName string) (int64, error) {
	edgeOffset := int64(0)
	meta, err := consumer.GetMetadata(&topicName, false, -1)
	if err != nil {
		return 0, err
	}

	topicMeta := meta.Topics[topicName]
	for _, partition := range topicMeta.Partitions {
		_, lastPartitionOffset, err := consumer.QueryWatermarkOffsets(topicName, partition.ID, -1)
		if err != nil {
			return 0, err
		}
		if lastPartitionOffset > edgeOffset {
			edgeOffset = lastPartitionOffset
		}
	}
	return edgeOffset, nil
}

func LastAndEdgeOffsetsByRunConsumer(consumer *kafka.Consumer, topicName string) (lastOffsetForRestoring int64, edgeOffset int64, err error) {
	t, err := getNextWritingOffsetByMetaData(consumer, topicName)
	if err != nil {
		return 0, 0, err
	}
	if t == 0 {
		return 0, 0, nil
	}
	metaData, err := consumer.GetMetadata(&topicName, false, 1000)
	if err != nil {
		return 0, 0, err
	}
	tp := kafka.TopicPartition{
		Topic:     &topicName,
		Partition: metaData.Topics[topicName].Partitions[0].ID,
	}

	offsets, err := consumer.Committed([]kafka.TopicPartition{tp}, 10000)
	if err != nil {
		return 0, 0, err
	}
	nextOffset := offsets[0].Offset
	if nextOffset < 0 {
		return 0, 0, nil
	}

	tp = kafka.TopicPartition{
		Topic:     &topicName,
		Partition: metaData.Topics[topicName].Partitions[0].ID,
		Offset:    kafka.OffsetTail(2),
	}
	err = consumer.Assign([]kafka.TopicPartition{tp})

	if err != nil {
		return 0, 0, fmt.Errorf("assigning last message: %w", err)
	}
	ch := consumer.Events()
	var msg *kafka.Message
	for msg == nil {
		ev := <-ch
		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
		default:
			return 0, 0, fmt.Errorf("recieve not handled event %s", e.String())
		}
	}

	lastOffsetAtTopic := msg.TopicPartition.Offset
	if nextOffset > lastOffsetAtTopic {
		return int64(lastOffsetAtTopic), 0, nil
	}
	return 0, int64(nextOffset), nil
}

func LastOffsetByNewConsumer(consumer *kafka.Consumer, topicName string) (lastOffsetForRestoring int64, err error) {
	t, err := getNextWritingOffsetByMetaData(consumer, topicName)
	if err != nil {
		return 0, err
	}
	if t == 0 {
		return 0, nil
	}
	metaData, err := consumer.GetMetadata(&topicName, false, 1000)
	if err != nil {
		return 0, err
	}

	tp := kafka.TopicPartition{
		Topic:     &topicName,
		Partition: metaData.Topics[topicName].Partitions[0].ID,
		Offset:    kafka.OffsetTail(2),
	}
	err = consumer.Assign([]kafka.TopicPartition{tp})
	if err != nil {
		return 0, fmt.Errorf("assigning last message: %w", err)
	}
	ch := consumer.Events()
	var msg *kafka.Message
	ev := <-ch
	switch e := ev.(type) {
	case *kafka.Message:
		msg = e
	default:
		return 0, fmt.Errorf("recieve not handled event %s", e.String())
	}
	lastOffsetAtTopic := msg.TopicPartition.Offset
	return int64(lastOffsetAtTopic), nil
}

func resetConsumerToBeginning(consumer *kafka.Consumer, topicName string) error {
	metaData, err := consumer.GetMetadata(&topicName, false, 1000)
	if err != nil {
		return err
	}
	tp := kafka.TopicPartition{
		Topic:     &topicName,
		Partition: metaData.Topics[topicName].Partitions[0].ID,
		Offset:    kafka.OffsetBeginning,
	}
	err = consumer.Assign([]kafka.TopicPartition{tp})
	if err != nil {
		return fmt.Errorf("assigning first message: %w", err)
	}
	return nil
}
