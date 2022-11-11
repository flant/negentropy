package io

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type testCase struct {
	countFilling  int64
	countMainRead int64
	lastOffset    int64
	edgeOffset    int64
	description   string
}

var testcases = []testCase{
	{0, 0, 0, 0, "reading from empty topic"},
	{10, 0, 0, 0, "reading from not empty topic, without main reading before"},
	{10, 5, 0, 5, "reading from topic, with partial main reading before"},
	{10, 10, 9, 0, "reading from topic, with full main reading before"},
	// {1000, 1000, 999, 0, "reading from topic, with full main reading before"},
}

func TestTableForLastAndEdgeOffsets(t *testing.T) {
	if os.Getenv("KAFKA_ON_LOCALHOST") != "true" {
		t.Skip("manual or integration test. Requires kafka")
	}
	sharedkafka.DoNotEncrypt = true // TODO remove

	for i := range testcases {
		testcase := testcases[i]
		fmt.Printf("====%#v===\n", testcase)
		broker := initializesMessageBroker(t)
		topic := broker.PluginConfig.SelfTopicName

		err := tryWithTimeOut(5, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				err := broker.CreateTopic(context.TODO(), topic, nil)
				assert.NoError(t, err, testcase.description+":creating topic")
				c <- struct{}{}
			}()
			return c
		})
		assert.NoError(t, err, testcase.description+":creating topic")

		err = tryWithTimeOut(5, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				fillTopic(t, broker, testcase.countFilling)
				c <- struct{}{}
			}()
			return c
		})
		assert.NoError(t, err, testcase.description+":filling messages")
		mainReadingGroupID := "group" + strings.ReplaceAll(time.Now().String()[11:19], ":", "_")

		timeout := 30
		if testcase.countMainRead > 100 {
			timeout = 100
		}
		err = tryWithTimeOut(timeout, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				mainConsumer, err := broker.GetSubscribedRunConsumer(mainReadingGroupID, topic)
				assert.NoError(t, err, testcase.description+":creating run consumer")
				mainRead(t, broker, mainConsumer, testcase.countMainRead)
				go func() {
					mainConsumer.Close()
				}()
				c <- struct{}{}
			}()
			return c
		})
		assert.NoError(t, err, testcase.description+":main reading messages")

		start := time.Now()
		messageCounter := int64(0)
		maxMessageCounter := testcase.countMainRead
		err = tryWithTimeOut(50, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				newConsumer, err := broker.GetRestorationReader()
				assert.NoError(t, err, testcase.description+":getting restoration reader")
				runConsumer, err := broker.GetUnsubscribedRunConsumer(mainReadingGroupID)
				assert.NoError(t, err, testcase.description+":getting unsubscribed consumer")
				err = RunRestorationLoop(newConsumer,
					runConsumer,
					topic,
					nil,
					func(_ *memdb.Txn, msg *kafka.Message, _ hclog.Logger) error {
						if string(msg.Value) != fmt.Sprintf("%d", messageCounter) ||
							messageCounter > maxMessageCounter {
							t.Errorf("expected:%d, got:%s", messageCounter, string(msg.Value))
							return nil
						}
						messageCounter++
						return nil
					},
					hclog.NewNullLogger())
				if messageCounter != maxMessageCounter {
					t.Errorf("expected last read:%d, got:%d", maxMessageCounter, messageCounter)
				}
				assert.NoError(t, err, testcase.description+":calling  RunRestorationLoop")
				go func() {
					newConsumer.Close()
					runConsumer.Close()
				}()
				c <- struct{}{}
			}()
			return c
		})
		if testcase.countMainRead > 0 {
			fmt.Printf("restoration read for %d messages, spent %s\n", testcase.countMainRead, time.Since(start).String())
		}
		assert.NoError(t, err, testcase.description+":restore reading  messages")

		err = tryWithTimeOut(50, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				runConsumer, err := broker.GetUnsubscribedRunConsumer(mainReadingGroupID)
				assert.NoError(t, err, "getting unsubscribed consumer")
				newConsumer, err := broker.GetRestorationReader()
				assert.NoError(t, err, "getting restoration reader")
				lastOffset, edgeOffset, _, err := LastAndEdgeOffsetsByRunConsumer(runConsumer, newConsumer, topic)
				assert.NoError(t, err, testcase.description+":call LastAndEdgeOffsetsByRunConsumers")
				assert.Equal(t, testcase.lastOffset, lastOffset)
				assert.Equal(t, testcase.edgeOffset, edgeOffset)

				lastOffsetAtTopic, _, err := LastOffsetByNewConsumer(newConsumer, topic)
				assert.NoError(t, err, testcase.description+":call LastAndEdgeOffsetsByRunConsumers")
				if testcase.countFilling == 0 {
					assert.Equal(t, testcase.countFilling, lastOffsetAtTopic)
				} else {
					assert.Equal(t, testcase.countFilling-1, lastOffsetAtTopic)
				}
				c <- struct{}{}
				go func() {
					runConsumer.Close()
					newConsumer.Close()
				}()
			}()
			return c
		})
		assert.NoError(t, err, testcase.description+":check offset functions")

		broker.DeleteTopic(context.TODO(), topic) // nolint:errcheck
		broker.Close()
		if t.Failed() {
			fmt.Println("testcase is FAILED ")
		} else {
			fmt.Println("testcase is PASSED ")
		}
	}
}

func TestLastOffsetByNewConsumer(t *testing.T) {
	if os.Getenv("KAFKA_ON_LOCALHOST") != "true" {
		t.Skip("manual or integration test. Requires kafka")
	}
	sharedkafka.DoNotEncrypt = true // TODO remove
	broker := initializesMessageBroker(t)
	topic := broker.PluginConfig.SelfTopicName
	err := broker.CreateTopic(context.TODO(), topic, nil)
	assert.NoError(t, err, "creating topic")
	fillTopic(t, broker, 15)

	newConsumer, err := broker.GetRestorationReader()
	assert.NoError(t, err, "getting restoration reader")
	l, p, err := LastOffsetByNewConsumer(newConsumer, topic)

	require.NoError(t, err, "LastOffsetByNewConsumer")
	assert.Equal(t, int64(14), l)
	assert.Equal(t, int32(0), p)
}

func TestLastOffsetByNewConsumerEmptyTopic(t *testing.T) {
	if os.Getenv("KAFKA_ON_LOCALHOST") != "true" {
		t.Skip("manual or integration test. Requires kafka")
	}
	sharedkafka.DoNotEncrypt = true // TODO remove
	broker := initializesMessageBroker(t)
	topic := broker.PluginConfig.SelfTopicName
	err := broker.CreateTopic(context.TODO(), topic, nil)
	assert.NoError(t, err, "creating topic")

	newConsumer, err := broker.GetRestorationReader()
	assert.NoError(t, err, "getting restoration reader")
	l, p, err := LastOffsetByNewConsumer(newConsumer, topic)

	require.NoError(t, err, "LastOffsetByNewConsumer")
	assert.Equal(t, int64(0), l)
	assert.Equal(t, int32(0), p)
}

func tryWithTimeOut(seconds int, runner func() <-chan struct{}) error {
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	c := runner()
	select {
	case <-timer.C:
		return fmt.Errorf("timeout is execeded")
	case <-c:
		return nil
	}
}

func fillTopic(t *testing.T, mb *sharedkafka.MessageBroker, n int64) {
	msgs := []sharedkafka.Message{}
	for i := 0; i < int(n); i++ {
		k := fmt.Sprintf("%d", i)
		msgs = append(msgs, sharedkafka.Message{
			Topic:   mb.PluginConfig.SelfTopicName,
			Key:     k,
			Value:   []byte(k),
			Headers: map[string][]byte{"k1": []byte(k)},
		})
	}
	err := mb.SendMessages(msgs, nil)
	if err != nil {
		t.Errorf("filling topic:%s", err.Error())
		return
	}
}

func mainRead(t *testing.T, mb *sharedkafka.MessageBroker, mainConsumer *kafka.Consumer, n int64) {
	ch := mainConsumer.Events()
	counter := 0
	for counter < int(n) {
		var msg *kafka.Message
		ev := <-ch
		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
			source, err := sharedkafka.NewSourceInputMessage(mainConsumer, msg.TopicPartition)
			if err != nil {
				panic(err)
			}

			err = mb.SendMessages(nil, source)
			require.NoError(t, err)
			if string(msg.Value) != fmt.Sprintf("%d", counter) {
				t.Errorf("expected:%d, got:%s", counter, string(msg.Value))
				return
			}
			counter++
		default:
			t.Errorf("receive not handled event %s", e.String())
			return
		}
	}
}

const serverKafka = "localhost:9094"

func initializesMessageBroker(t *testing.T) *sharedkafka.MessageBroker {
	storage := &logical.InmemStorage{}
	key := "kafka.config"
	pl := "kafka.plugin.config"

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	config := sharedkafka.BrokerConfig{
		Endpoints:            []string{serverKafka},
		EncryptionPrivateKey: pk,
		EncryptionPublicKey:  &pk.PublicKey,
	}

	plugin := sharedkafka.PluginConfig{
		SelfTopicName: "topic_" + strings.ReplaceAll(time.Now().String()[11:23], ":", "_"),
	}
	d1, _ := json.Marshal(config)
	d2, _ := json.Marshal(plugin)
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: key, Value: d1})
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: pl, Value: d2})

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), storage, hclog.NewNullLogger())
	require.NoError(t, err)
	return mb
}
