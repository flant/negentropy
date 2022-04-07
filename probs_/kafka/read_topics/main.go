package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	kafka2 "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const serverKafka = "localhost:9093"

var topics = []string{
	"root_source", "auth-source.auth-1", "auth-source.auth-2", "jwks",
	"multipass_generation_num", "root_source.auth-1", "root_source.auth-2",
}

func main() {
	for _, topic := range topics {
		types := collectTypesFromTopic(topic)
		fmt.Printf("\ntopic: %s\n===================\n", topic)
		for k, v := range types {
			fmt.Printf("%s - %d\n", k, v)
		}
	}
}

func collectTypesFromTopic(topic string) map[string]int {
	result := map[string]int{}
	mb, err := messageBroker()
	if err != nil {
		panic(err)
	}
	groupID := "group" + strings.Replace(time.Now().String()[11:19], ":", "_", -1)
	runConsumer := mb.GetUnsubscribedRunConsumer(groupID)
	l, _, err := kafka2.LastOffsetByNewConsumer(runConsumer, topic)
	if err != nil {
		panic(err)
	}
	runConsumer.Unassign()
	runConsumer.Subscribe(topic, nil)

	ch := runConsumer.Events()
loop:
	for {
		ev := <-ch
		switch e := ev.(type) {
		case *kafka.Message:
			split := strings.Split(string(e.Key), "/")

			result[split[0]] = result[split[0]] + 1
			// fmt.Printf("key %s, offset %d, t %t\n", split[0], e.TopicPartition.Offset, int64(e.TopicPartition.Offset) == l)
			if int64(e.TopicPartition.Offset) == l {
				break loop
			}
		default:
			fmt.Printf("Recieve not handled event %s", e.String())
			continue
		}
	}
	return result
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
