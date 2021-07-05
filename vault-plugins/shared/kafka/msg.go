package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type MessageHandler func(sourceConsumer *kafka.Consumer, store *io.MemoryStore, msg *kafka.Message)

func RunMessageLoop(c *kafka.Consumer, store *io.MemoryStore, msgHandler MessageHandler, stopC chan struct{}) {
	for {
		select {
		case <-stopC:
			c.Unsubscribe()
			c.Close()
			return

		case ev := <-c.Events():
			switch e := ev.(type) {
			case *kafka.Message:
				msgHandler(c, store, e)

			default:
				// ignore other events
			}
		}
	}
}

func RunRestorationLoop(newConsumer, runConsumer *kafka.Consumer, topicName string, txn *memdb.Txn, handler func(txn *memdb.Txn, msg *kafka.Message) error) error {
	var lastOffset int64

	if runConsumer != nil {
		// get the latest offset from existing reader
		tp, err := runConsumer.Assignment()
		if err != nil {
			return err
		}
		offsets, err := runConsumer.Position(tp)
		if err != nil {
			return err
		}

		for _, offset := range offsets {
			if offset.Topic == nil || *offset.Topic != topicName {
				continue
			}

			if lastOffset < int64(offset.Offset) {
				lastOffset = int64(offset.Offset)
			}
		}
	} else {
		// else get lastOffset from topic - for self topics
		meta, err := newConsumer.GetMetadata(&topicName, false, -1)
		if err != nil {
			return err
		}

		topicMeta := meta.Topics[topicName]
		for _, partition := range topicMeta.Partitions {
			_, lastPartitionOffset, err := newConsumer.QueryWatermarkOffsets(topicName, partition.ID, -1)
			if err != nil {
				return err
			}
			if lastPartitionOffset > lastOffset {
				lastOffset = lastPartitionOffset
			}
		}
	}

	if lastOffset <= 0 {
		return nil
	}

	for {
		msg, err := newConsumer.ReadMessage(-1)
		if err != nil {
			return err
		}

		err = handler(txn, msg)
		if err != nil {
			return err
		}

		if int64(msg.TopicPartition.Offset) == lastOffset-1 {
			return nil
		}
	}
}
