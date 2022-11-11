// Implements ways to store&extract lastOffset for RestorationLoop

package io

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/vault/sdk/logical"
)

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
	for msg == nil {
		ev := <-ch
		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
		default:
			fmt.Printf("Collected from topic %s unsupported event: %#v\n", topicName, ev)
		}
	}
	lastOffsetAtTopic := msg.TopicPartition.Offset
	return int64(lastOffsetAtTopic), partition, nil
}

func lastOffsetStorageKey(runConsumerGroupID string, topicName string) string {
	// use this combination to have unique at vault instance
	return fmt.Sprintf("%s-%s-offsetStoraegKey", runConsumerGroupID, topicName)
}

// LastOffsetFromStorage returns -1 if nothing is  stored
func LastOffsetFromStorage(ctx context.Context, storage logical.Storage, runConsumerGroupID string,
	topicName string) (lastOffsetForRestoring int64, err error) {
	key := lastOffsetStorageKey(runConsumerGroupID, topicName)
	se, err := storage.Get(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("getting last offset from storage: %w", err)
	}
	if se == nil {
		return -1, nil
	}
	if len(se.Value) != 8 {
		return 0, fmt.Errorf("wrong len of data: %d", len(se.Value))
	}
	lastOffsetForRestoring = int64(binary.BigEndian.Uint64(se.Value))
	return
}

func StoreLastOffsetToStorage(ctx context.Context, storage logical.Storage, runConsumerGroupID string,
	topicName string, lastOffsetForStoring int64) (err error) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(lastOffsetForStoring))

	err = storage.Put(ctx, &logical.StorageEntry{
		Key:   lastOffsetStorageKey(runConsumerGroupID, topicName),
		Value: b,
	})
	if err != nil {
		return fmt.Errorf("putting last offset to storage: %w", err)
	}
	return
}
