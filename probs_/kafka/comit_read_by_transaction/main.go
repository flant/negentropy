package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	kafka2 "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const serverKafka = "localhost:9093"

var groupID = "group" + strings.Replace(time.Now().String()[11:19], ":", "_", -1)

func main() {
	mb, err := messageBroker()
	if err != nil {
		panic(err)
	}
	topicName := mb.PluginConfig.SelfTopicName
	fmt.Printf("topic = %s\n", topicName)

	//	и посмотрим

	fillMyTopic(mb)

	// На С0 попробуем получить ноль
	runConsumer := mb.GetUnsubscribedRunConsumer(groupID)
	newConsumer := mb.GetRestorationReader()
	l, e, _ := LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topicName)
	fmt.Printf(" need '0 0', got %d %d\n", l, e)
	// runConsumer.Close()

	//// read some portion
	println(runConsumer.Subscribe(topicName, nil))
	readNMessagesByConsumer(mb, runConsumer, topicName, groupID, 3)
	println(time.Now().String())

	kafka2.RunRestorationLoop(newConsumer, runConsumer, topicName, nil, func(_ *memdb.Txn, msg *kafka.Message, log hclog.Logger) error {
		fmt.Printf("====>%s\n", msg.Value)
		return nil
	}, hclog.Default())
	println(time.Now().String())

	// c2 := mb.GetUnsubscribedRunConsumer(groupID)
	l, e, _ = LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topicName)
	fmt.Printf(" need '0 5', got %d %d\n", l, e)
	println(time.Now().String())

	// RebalanceCb provides a per-Subscribe*() rebalance event callback.
	// The passed Event will be either AssignedPartitions or RevokedPartitions
	// type RebalanceCb func(*Consumer, Event) error

	readNMessagesByConsumer(mb, runConsumer, topicName, groupID, 4)
	println(time.Now().String())

	l, e, _ = LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topicName)
	fmt.Printf(" need '12 0', got %d %d\n", l, e)

	fmt.Println("======================")
}

func fillMyTopic(mb *kafka2.MessageBroker) {
	mb.CreateTopic(context.TODO(), mb.PluginConfig.SelfTopicName, nil)

	// Produce messages to topic (asynchronously)
	// msgs := []kafka2.Message{}
	for _, word := range []string{"1_Welcome", "2_to", "3_the", "4_Confluent", "5_Kafka", "6_Golang", "7_client"} {
		err := mb.SendMessages([]kafka2.Message{
			{
				Topic:   mb.PluginConfig.SelfTopicName,
				Key:     "word",
				Value:   []byte(word),
				Headers: nil,
			},
		}, nil)
		if err != nil {
			panic(err)
		}

	}
}

func messageBroker() (*kafka2.MessageBroker, error) {
	storage := &logical.InmemStorage{}
	key := "kafka.config"
	pl := "kafka.plugin.config"

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	config := kafka2.BrokerConfig{
		Endpoints:             []string{serverKafka},
		ConnectionPrivateKey:  nil,
		ConnectionCertificate: nil,
		EncryptionPrivateKey:  pk,
		EncryptionPublicKey:   &pk.PublicKey,
	}

	plugin := kafka2.PluginConfig{
		SelfTopicName: "topic_" + strings.ReplaceAll(time.Now().String()[11:23], ":", "_"),
	}
	d1, _ := json.Marshal(config)
	d2, _ := json.Marshal(plugin)
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: key, Value: d1})
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: pl, Value: d2})

	mb, err := kafka2.NewMessageBroker(context.TODO(), storage, hclog.New(&hclog.LoggerOptions{
		Name:            "test",
		IncludeLocation: true,
	}))
	if err != nil {
		return nil, err
	}

	return mb, nil
}

func readNMessagesByConsumer(mb *kafka2.MessageBroker, c *kafka.Consumer, topicName string, groupID string, n int) {
	ch := c.Events()
	counter := 0
	for {
		counter++
		if counter > n {
			break
		}
		var msg *kafka.Message
		ev := <-ch

		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
			source, err := kafka2.NewSourceInputMessage(c, msg.TopicPartition)
			if err != nil {
				panic(err)
			}
			mb.SendMessages(nil, source)

		default:
			fmt.Printf("Recieve not handled event %s", e.String())
			continue
		}
		fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
	}
	// c.Close()
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

func LastAndEdgeOffsetsByRunConsumer(runConsumer *kafka.Consumer, newConsumer *kafka.Consumer,
	topicName string) (lastOffsetForRestoring int64, edgeOffset int64, err error) {
	t, err := getNextWritingOffsetByMetaData(runConsumer, topicName)
	if err != nil {
		return 0, 0, err
	}
	if t == 0 {
		return 0, 0, nil
	}
	metaData, err := runConsumer.GetMetadata(&topicName, false, 1000)
	if err != nil {
		return 0, 0, err
	}
	tp := kafka.TopicPartition{
		Topic:     &topicName,
		Partition: metaData.Topics[topicName].Partitions[0].ID,
	}

	offsets, err := runConsumer.Committed([]kafka.TopicPartition{tp}, 10000)
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
	err = newConsumer.Assign([]kafka.TopicPartition{tp})

	if err != nil {
		return 0, 0, fmt.Errorf("assigning last message: %w", err)
	}
	ch := newConsumer.Events()
	var msg *kafka.Message
	for msg == nil {
		ev := <-ch
		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
			// default:
			//	return 0, 0, fmt.Errorf("recieve not handled event %s", e.String())
		}
	}

	lastOffsetAtTopic := msg.TopicPartition.Offset
	if nextOffset > lastOffsetAtTopic {
		return int64(lastOffsetAtTopic), 0, nil
	}
	return 0, int64(nextOffset), nil
}
