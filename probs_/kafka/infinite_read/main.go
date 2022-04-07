package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

const serverKafka = "localhost:9093"

var (
	groupID   = "group" + strings.Replace(time.Now().String()[11:19], ":", "_", -1)
	topicName = "i_" + strings.Replace(time.Now().String()[11:19], ":", "_", -1)
)

func main() {
	fmt.Printf("topic = %s", topicName)

	fillMyTopic()

	// На С0 попробуем получить ноль
	c0 := consumer()
	println(LastAndEdgeOffsets(c0, topicName))
	c0.Close()

	//// read some portion
	c1 := consumer()
	readNMessages(c1, 3)
	c1.Close()

	c2 := consumer()
	println(LastAndEdgeOffsets(c2, topicName))
	c2.Close()

	c3 := consumer()
	readNMessages(c3, 4)
	c3.Close()

	c4 := consumer()
	println(LastAndEdgeOffsets(c4, topicName))
	c4.Close()

	fmt.Println("======================")
}

func readNMessages(c *kafka.Consumer, n int) {
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
			c.CommitMessage(msg)
		default:
			fmt.Printf("Recieve not handled event %s", e.String())
			continue
		}
		fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
	}
}

func consumer() *kafka.Consumer {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        serverKafka,
		"group.id":                 groupID,
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       true,
		"isolation.level":          "read_committed",
		"go.events.channel.enable": true,
	})
	if err != nil {
		panic(err)
	}
	c.SubscribeTopics([]string{topicName}, nil)
	return c
}

func fillMyTopic() {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": serverKafka})
	if err != nil {
		panic(err)
	}

	defer p.Close()

	// Delivery report handler for produced messages
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}()

	// Produce messages to topic (asynchronously)
	for _, word := range []string{"Welcome", "to", "the", "Confluent", "Kafka", "Golang", "client"} {
		p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topicName, Partition: kafka.PartitionAny},
			Value:          []byte(word),
		}, nil)
	}

	// Wait for message deliveries before shutting down
	p.Flush(15 * 1000)
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

func LastAndEdgeOffsets(consumer *kafka.Consumer, topicName string) (lastOffsetForRestoring int64, edgeOffset int64, err error) {
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

	offsets, err := consumer.Committed([]kafka.TopicPartition{tp}, 1000)
	nextOffset := offsets[0].Offset
	if nextOffset < 0 {
		return 0, 0, nil
	}
	tp = kafka.TopicPartition{
		Topic:     &topicName,
		Partition: metaData.Topics[topicName].Partitions[0].ID,
		Offset:    kafka.OffsetTail(1),
	}
	consumer.Assign([]kafka.TopicPartition{tp})
	ch := consumer.Events()
	var msg *kafka.Message
	ev := <-ch
	switch e := ev.(type) {
	case *kafka.Message:
		msg = e
	default:
		return 0, 0, fmt.Errorf("recieve not handled event %s", e.String())
	}
	lastOffsetAtTopic := msg.TopicPartition.Offset
	if nextOffset > lastOffsetAtTopic {
		return int64(lastOffsetAtTopic), 0, nil
	}
	return 0, int64(nextOffset), nil
}
