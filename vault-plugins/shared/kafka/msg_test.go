package kafka

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const serverKafka = "localhost:9093"

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
}

func TestTable(t *testing.T) {
	DoNotEncrypt = true // TODO remove
	for i := range testcases {
		testcase := testcases[i]
		fmt.Printf("====%#v===\n", testcase)
		broker := messageBroker(t)
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

		err = tryWithTimeOut(30, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				mainRead(t, broker, topic, mainReadingGroupID, testcase.countMainRead)
				c <- struct{}{}
			}()
			return c
		})
		assert.NoError(t, err, testcase.description+":main reading messages")

		messageCounter := int64(0)
		maxMessageCounter := testcase.countMainRead
		err = tryWithTimeOut(50, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				newConsumer := broker.GetRestorationReader()
				runConsumer := broker.GetUnsubscribedRunConsumer(mainReadingGroupID)
				err := RunRestorationLoop(newConsumer,
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
				newConsumer.Close()
				runConsumer.Close()
				c <- struct{}{}
			}()
			return c
		})

		assert.NoError(t, err, testcase.description+":restore reading  messages")

		err = tryWithTimeOut(50, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				runConsumer := broker.GetUnsubscribedRunConsumer(mainReadingGroupID)
				lastOffset, edgeOffset, err := LastAndEdgeOffsetsByRunConsumer(runConsumer, topic)
				assert.NoError(t, err, testcase.description+":call LastAndEdgeOffsetsByRunConsumers")
				assert.Equal(t, testcase.lastOffset, lastOffset)
				assert.Equal(t, testcase.edgeOffset, edgeOffset)

				newConsumer := broker.GetRestorationReader()
				lastOffsetAtTopic, err := LastOffsetByNewConsumer(newConsumer, topic)
				assert.NoError(t, err, testcase.description+":call LastAndEdgeOffsetsByRunConsumers")
				if testcase.countFilling == 0 {
					assert.Equal(t, testcase.countFilling, lastOffsetAtTopic)
				} else {
					assert.Equal(t, testcase.countFilling-1, lastOffsetAtTopic)
				}
				c <- struct{}{}
				runConsumer.Close()
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

func fillTopic(t *testing.T, mb *MessageBroker, n int64) {
	msgs := []Message{}
	for i := 0; i < int(n); i++ {
		k := fmt.Sprintf("%d", i)
		msgs = append(msgs, Message{
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

func mainRead(t *testing.T, mb *MessageBroker, topic string, mainReadingGroupID string, n int64) {
	mainConsumer := mb.GetSubscribedRunConsumer(mainReadingGroupID, topic)
	ch := mainConsumer.Events()
	counter := 0
	for counter < int(n) {
		var msg *kafka.Message
		ev := <-ch
		switch e := ev.(type) {
		case *kafka.Message:
			msg = e
			_, err := mainConsumer.CommitMessage(msg)
			if err != nil {
				t.Errorf("unexpected error:%s", err.Error())
				return
			}
			if string(msg.Value) != fmt.Sprintf("%d", counter) {
				t.Errorf("expected:%d, got:%s", counter, string(msg.Value))
				return
			}
			counter++
		default:
			t.Errorf("receive not handled event %s", e.String())
			return
		}
		// fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
	}
	mainConsumer.Close()
}

func messageBroker(t *testing.T) *MessageBroker {
	storage := &logical.InmemStorage{}
	key := "kafka.config"
	pl := "kafka.plugin.config"

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	config := BrokerConfig{
		Endpoints:             []string{serverKafka},
		ConnectionPrivateKey:  nil,
		ConnectionCertificate: nil,
		EncryptionPrivateKey:  pk,
		EncryptionPublicKey:   &pk.PublicKey,
	}

	plugin := PluginConfig{
		SelfTopicName: "topic_" + strings.ReplaceAll(time.Now().String()[11:23], ":", "_"),
	}
	d1, _ := json.Marshal(config)
	d2, _ := json.Marshal(plugin)
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: key, Value: d1})
	err = storage.Put(context.TODO(), &logical.StorageEntry{Key: pl, Value: d2})

	mb, err := NewMessageBroker(context.TODO(), storage)
	require.NoError(t, err)
	return mb
}
