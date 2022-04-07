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
	topicName = "myTopic" + strings.Replace(time.Now().String()[11:19], ":", "_", -1)
)

func main() {
	fillMyTopic()

	// read some portion
	c1 := consumer()
	// readNMessages(c1, 3)
	readMessagesWithoutChannels(c1, 3)
	c1.Close()

	fmt.Printf("read 3 messages, last read offset = 2\n")
	fmt.Println()
	fmt.Printf("try use Assignment() & Position(tp):\n")
	c2 := consumer()
	e2 := c2.Poll(100)
	fmt.Printf("%#v <= we have got empty TopicPartion\n", e2)
	tp, err := c2.Assignment()
	fmt.Printf("%#v, %#v <= we have got empty TopicPartion\n", tp, err)
	offsets, err := c2.Position(tp)
	fmt.Printf("%#v, %#v <= we have got empty offsets\n", offsets, err)
	fmt.Println()
	fmt.Printf("try use one uncommited read and than Assignment() & Position(tp):\n")

	tp, err = c2.Assignment()
	fmt.Printf("%#v, %#v <= we have got not empty TopicPartion\n", tp, err)
	offsets, err = c2.Position(tp)
	fmt.Printf("%#v, %#v <= we have got not empty offsets\n", offsets, err)
	fmt.Printf("but offset is wrong: got %d, need 3\n", offsets[0].Offset)
	fmt.Println()

	ch := c2.Events()
	e := <-ch
	msg := e.(*kafka.Message)
	fmt.Printf("read offset from message: %d and close uncommited\n", msg.TopicPartition.Offset)
	c2.Close()

	// read last protion
	c3 := consumer()

	readNMessages(c3, 4)
	c3.Close()

	fmt.Println()
	c4 := consumerGroupFalse()
	readNMessagesWithoutCommit(c4, 7)
	c4.Close()

	fmt.Println()
	c5 := consumerGroupFalse()
	readNMessagesWithoutCommit(c5, 7)
	c5.Close()
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

func readMessagesWithoutChannels(c *kafka.Consumer, n int) {
	counter := 0
	for {
		counter++
		if counter > n {
			break
		}
		msg, err := c.ReadMessage(-1)
		if err != nil {
			panic(err)
		}
		c.CommitMessage(msg)
		fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
	}
}

func consumer() *kafka.Consumer {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        serverKafka,
		"group.id":                 groupID,
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       false,
		"isolation.level":          "read_committed",
		"go.events.channel.enable": false,
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

func consumerGroupFalse() *kafka.Consumer {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        serverKafka,
		"group.id":                 false,
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       false,
		"isolation.level":          "read_committed",
		"go.events.channel.enable": true,
	})
	if err != nil {
		panic(err)
	}
	c.SubscribeTopics([]string{topicName}, nil)
	return c
}

func readNMessagesWithoutCommit(c *kafka.Consumer, n int) {
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
		default:
			fmt.Printf("Recieve not handled event %s", e.String())
			continue
		}
		fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
	}
}
