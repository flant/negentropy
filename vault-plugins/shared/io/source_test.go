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

	"github.com/hashicorp/go-hclog"
	hcmemdb "github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type testCase struct {
	countFilling  int64
	countMainRead int64
	description   string
}

var testcases = []testCase{
	{0, 0, "reading from empty topic"},
	{10, 0, "reading from not empty topic, without main reading before"},
	{10, 5, "reading from topic, with partial main reading before"},
	{10, 10, "reading from topic, with full main reading before"},
	// {1000, 1000, 999, 0, "reading from topic, with full main reading before"},
}

func TestKafkaSourceRunAndRestore(t *testing.T) {
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
		runConsumerGroupID := "group" + strings.ReplaceAll(time.Now().String()[11:19], ":", "_")

		start := time.Now()
		ms := simpleMemstore(broker)
		//maxMessageCounter := testcase.countMainRead
		runConsumerCounter := int64(0)
		mainReadDoneChan := make(chan struct{})
		restoreConsumerCounter := int64(0)
		vaultStorage := &logical.InmemStorage{}
		err = tryWithTimeOut(50, func() <-chan struct{} {
			c := make(chan struct{})
			go func() {
				defer func() {
					c <- struct{}{}
				}()
				ks := KafkaSourceImpl{
					NameOfSource: "test",
					KafkaBroker:  broker,
					Logger:       hclog.Default(),
					ProvideRunConsumerGroupID: func(kf *sharedkafka.MessageBroker) string {
						return runConsumerGroupID
					},
					ProvideTopicName: func(kf *sharedkafka.MessageBroker) string {
						return topic
					},
					VerifySign: func(signature []byte, messageValue []byte) error {
						return nil
					},
					Decrypt: func(encryptedMessageValue []byte, chunked bool) ([]byte, error) {
						return encryptedMessageValue, nil
					},
					ProcessRunMessage: func(txn Txn, m MsgDecoded) error {
						if string(m.Data) != fmt.Sprintf("%d", runConsumerCounter) {
							t.Errorf("expected:%d, got:%s", runConsumerCounter, string(m.Data))
						}
						runConsumerCounter++
						if runConsumerCounter == testcase.countMainRead {
							mainReadDoneChan <- struct{}{}
						}
						return nil
					},
					ProcessRestoreMessage: func(txn Txn, m MsgDecoded) error {
						if string(m.Data) != fmt.Sprintf("%d", restoreConsumerCounter) {
							t.Errorf("expected:%d, got:%s", restoreConsumerCounter, string(m.Data))
						}
						restoreConsumerCounter++
						return nil
					},
					IgnoreSourceInputMessageBody:    true,
					Runnable:                        true,
					SkipRestorationOnWrongSignature: false,
					RestoreStrictlyTillRunConsumer:  true,
					Storage:                         vaultStorage,
				}
				if testcase.countMainRead > 0 {
					go func() {
						ks.Run(ms)
					}()

					<-mainReadDoneChan
					ks.Stop()
				}

				expectedOffset := int64(-1)
				if testcase.countMainRead > 0 {
					expectedOffset = runConsumerCounter - 1
				}

				storedLastOffset, err := LastOffsetFromStorage(context.Background(), vaultStorage, runConsumerGroupID, topic)
				assert.NoError(t, err, testcase.description+": getting stored offset")

				if storedLastOffset != expectedOffset {
					t.Errorf("expected stored offset: %d, got:%d", expectedOffset, storedLastOffset)
				}

				err = ks.Restore(ms.Txn(true).Txn)
				assert.NoError(t, err, testcase.description+":calling  Restore")
				if restoreConsumerCounter != runConsumerCounter {
					t.Errorf("expected last read:%d, got:%d", runConsumerCounter, restoreConsumerCounter)
				}
			}()
			return c
		})

		assert.NoError(t, err, testcase.description+": running ks staff")

		if testcase.countMainRead > 0 {
			fmt.Printf("restoration read for %d messages, spent %s\n", testcase.countMainRead, time.Since(start).String())
		}

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

func fillTopic(t *testing.T, mb *sharedkafka.MessageBroker, n int64) {
	msgs := []sharedkafka.Message{}
	for i := 0; i < int(n); i++ {
		k := fmt.Sprintf("%d", i)
		key := "obj/" + k
		msgs = append(msgs, sharedkafka.Message{
			Topic:   mb.PluginConfig.SelfTopicName,
			Key:     key,
			Value:   []byte(k),
			Headers: map[string][]byte{"signature": []byte("fake_signature")},
		})
	}
	err := mb.SendMessages(msgs, nil)
	if err != nil {
		t.Errorf("filling topic:%s", err.Error())
		return
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
		SSLConfig: &sharedkafka.SSLConfig{
			UseSSL:                true,
			CAPath:                "../../../docker/kafka/ca.crt",
			ClientPrivateKeyPath:  "../../../docker/kafka/client.key",
			ClientCertificatePath: "../../../docker/kafka/client.crt",
		},
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

func simpleMemstore(kb *sharedkafka.MessageBroker) *MemoryStore {
	ms, err := NewMemoryStore(
		&memdb.DBSchema{
			Tables: map[string]*memdb.TableSchema{"test": &memdb.TableSchema{
				Name: "test",
				Indexes: map[string]*hcmemdb.IndexSchema{"id": &hcmemdb.IndexSchema{
					Name:   "id",
					Unique: true,
					Indexer: &hcmemdb.StringFieldIndex{
						Field: "test",
					},
				}},
			}},
		}, kb, hclog.NewNullLogger(),
	)
	if err != nil {
		panic(err)
	}
	return ms
}
