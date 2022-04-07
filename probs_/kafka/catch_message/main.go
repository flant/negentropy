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
	topicName = "root_source.auth-1"
	checker   = func(message *kafka.Message) bool {
		if string(message.Key) == "entity/acaa947b-ac8f-41cb-8e6f-ca5088cd8874" {
			return true
		}
		if string(message.Key) == "entity_alias/0ad3afc1-f52d-4d02-ac36-0b143efa0226" {
			return true
		}
		if strings.Contains(string(message.Key), "3a44ef3d-c95a-456a-b622-e1dfc7e8a52f") {
			return true
		}
		return false
	}
)

func main() {
	// read some portion
	c1 := consumer()
	readMessages(c1, checker)
	c1.Close()
}

func readMessages(c *kafka.Consumer, checker func(message *kafka.Message) bool) {
	ch := c.Events()
	for {
		var msg *kafka.Message
		ev := <-ch

		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
			if checker(msg) {
				println("================================================")
				println(string(msg.Key))
				fmt.Printf("%#v", msg.Headers)
				println(string(msg.Value))
				println("================================================")
			}

		default:
			fmt.Printf("Recieve not handled event %s", e.String())
			continue
		}
	}
}

func consumer() *kafka.Consumer {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        serverKafka,
		"group.id":                 groupID,
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
