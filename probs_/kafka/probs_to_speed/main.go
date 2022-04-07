package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/profile"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	kafka2 "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const serverKafka = "localhost:9093"

var groupID = "group" + strings.Replace(time.Now().String()[11:19], ":", "_", -1)

func main() {
	defer profile.Start(profile.CPUProfile).Stop()
	tmp := time.Now()
	lastLogTime := &tmp
	mb, err := messageBroker()
	if err != nil {
		panic(err)
	}
	topicName := mb.PluginConfig.SelfTopicName
	fmt.Printf("topic = %s\n", topicName)
	logWithTime(lastLogTime, "Create producer")

	fillMyTopic(mb)
	logWithTime(lastLogTime, "Create topic, fill topic") // На С0 попробуем получить ноль

	runConsumer := mb.GetUnsubscribedRunConsumer(groupID)
	newConsumer := mb.GetRestorationReader()
	logWithTime(lastLogTime, "Create consumers")

	l, e, _ := LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topicName)
	fmt.Printf(" need '0 0', got %d %d\n", l, e)
	logWithTime(lastLogTime, "Call function LastAndEdgeOffsetsByRunConsumer")
	runConsumer.Subscribe(topicName, nil)

	//// read some portion
	readNMessagesByConsumer(mb, runConsumer, 3)
	logWithTime(lastLogTime, "read some messages")

	l, e, _ = LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topicName)
	fmt.Printf(" need '0 5', got %d %d\n", l, e)
	logWithTime(lastLogTime, "Call function LastAndEdgeOffsetsByRunConsumer")

	readNMessagesByConsumer(mb, runConsumer, 4)
	logWithTime(lastLogTime, "read some messages")

	l, e, _ = LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topicName)
	fmt.Printf(" need '12 0', got %d %d\n", l, e)
	logWithTime(lastLogTime, "Call function LastAndEdgeOffsetsByRunConsumer")

	logWithTime(lastLogTime, "Close consumer")

	fmt.Println("======================")
	mb.Close()
	logWithTime(lastLogTime, "Close mb")

	time.Sleep(100000)
}

func logWithTime(logTime *time.Time, message string) {
	now := time.Now()
	fmt.Printf("===>%s: "+message+"\n", now.Sub(*logTime).String())
	*logTime = now
}

func readNMessagesByConsumer(mb *kafka2.MessageBroker, c *kafka.Consumer, n int) {
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

func fillMyTopic(mb *kafka2.MessageBroker) {
	mb.CreateTopic(context.TODO(), mb.PluginConfig.SelfTopicName, nil)
	for _, word := range []string{"Welcome", "to", "the", "Confluent", "Kafka", "Golang", "client"} {
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
	topicName string) (lastOffsetForRestoring int64, edgeOffset int64, err error) {
	tmp := time.Now()
	lastLogTime := &tmp
	metaData, t, err := getNextWritingOffsetByMetaData(runConsumer, topicName)
	logWithTime(lastLogTime, "==== getNextWritingOffsetByMetaDatas")
	if err != nil {
		return 0, 0, err
	}
	if t == 0 {
		return 0, 0, nil
	}
	tp := kafka.TopicPartition{
		Topic:     &topicName,
		Partition: metaData.Topics[topicName].Partitions[0].ID,
	}

	offsets, err := runConsumer.Committed([]kafka.TopicPartition{tp}, 10000)
	logWithTime(lastLogTime, "==== Committed")

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
	logWithTime(lastLogTime, "==== Assign")
	defer newConsumer.Unassign()
	if err != nil {
		return 0, 0, fmt.Errorf("assigning last message: %w", err)
	}
	ch := newConsumer.Events()
	var msg *kafka.Message
	logWithTime(lastLogTime, "==== Events")
	for msg == nil {
		logWithTime(lastLogTime, "==== loop")
		ev := <-ch
		logWithTime(lastLogTime, "==== loop2")

		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
		default:
			return 0, 0, fmt.Errorf("recieve not handled event %s", e.String())
		}
	}
	logWithTime(lastLogTime, "==== for")

	lastOffsetAtTopic := msg.TopicPartition.Offset
	if nextOffset > lastOffsetAtTopic {
		return int64(lastOffsetAtTopic), 0, nil
	}
	return 0, int64(nextOffset), nil
}
