package kafka

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/vault/sdk/logical"
)

// TopicType represents kafka topic type
type TopicType string

func (t TopicType) String() string {
	return string(t)
}

type Message struct {
	Topic   string
	Key     string
	Value   []byte
	Headers map[string][]byte
}

// SourceInputMessage kafka message from consumer
type SourceInputMessage struct {
	TopicPartition   []kafka.TopicPartition
	ConsumerMetadata *kafka.ConsumerGroupMetadata
}

func NewSourceInputMessage(c *kafka.Consumer, tp kafka.TopicPartition) (*SourceInputMessage, error) {
	pos, err := c.Position([]kafka.TopicPartition{tp})
	if err != nil {
		return nil, err
	}
	cm, err := c.GetConsumerGroupMetadata()
	if err != nil {
		return nil, err
	}
	return &SourceInputMessage{
		TopicPartition:   pos,
		ConsumerMetadata: cm,
	}, nil
}

type MessageBroker struct {
	isConfigured bool

	producerSync      sync.Once
	producer          *kafka.Producer
	transProducerSync sync.Once
	transProducer     *kafka.Producer

	config       BrokerConfig
	PluginConfig PluginConfig
}

func NewMessageBroker(ctx context.Context, storage logical.Storage) (*MessageBroker, error) {
	mb := &MessageBroker{}

	// load encryption private key
	se, err := storage.Get(ctx, kafkaConfigPath)
	if err != nil {
		return nil, err
	}
	if se != nil {
		var config BrokerConfig

		err = json.Unmarshal(se.Value, &config)
		if err != nil {
			return nil, err
		}

		mb.config = config
	}

	se, err = storage.Get(ctx, PluginConfigPath)
	if err != nil {
		return nil, err
	}
	if se != nil {
		var config PluginConfig

		err = json.Unmarshal(se.Value, &config)
		if err != nil {
			return nil, err
		}

		mb.PluginConfig = config
	}

	mb.CheckConfig()

	return mb, nil
}

func (mb *MessageBroker) CheckConfig() {
	if len(mb.config.Endpoints) > 0 &&
		mb.config.EncryptionPublicKey != nil &&
		mb.config.EncryptionPrivateKey != nil &&
		mb.PluginConfig.SelfTopicName != "" {
		mb.isConfigured = true
	}
}

func (mb *MessageBroker) Configured() bool {
	return mb.isConfigured
}

func (mb *MessageBroker) CheckConnection(endpoints []string) error {
	brokers := strings.Join(endpoints, ",")
	ac, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"retries":           3,
	})
	if err != nil {
		return err
	}
	defer ac.Close()

	_, err = ac.ClusterID(context.TODO())
	return err
}

// BrokerConfig connection config
type BrokerConfig struct {
	Endpoints []string `json:"endpoints"`

	ConnectionPrivateKey  *ecdsa.PrivateKey `json:"connection_private_key,omitempty"`
	ConnectionCertificate *x509.Certificate `json:"connection_cert,omitempty"`

	EncryptionPrivateKey *rsa.PrivateKey `json:"encrypt_private_key,omitempty"`
	EncryptionPublicKey  *rsa.PublicKey  `json:"encrypt_public_key,omitempty"`
}

type unmarshalablePrivateKey ecdsa.PrivateKey

func (un unmarshalablePrivateKey) toECDSA() *ecdsa.PrivateKey {
	pk := ecdsa.PrivateKey(un)
	return &pk
}

func (un *unmarshalablePrivateKey) UnmarshalJSON(b []byte) error {
	var a ecdsa.PrivateKey
	_ = json.Unmarshal(b, &a) // cannot unmarshal only curve here

	*un = unmarshalablePrivateKey(a)
	un.Curve = elliptic.P256()

	return nil
}

func (bc *BrokerConfig) UnmarshalJSON(data []byte) error {
	s := struct {
		Endpoints []string `json:"endpoints"`

		ConnectionPrivateKey  unmarshalablePrivateKey `json:"connection_private_key,omitempty"`
		ConnectionCertificate *x509.Certificate       `json:"connection_cert,omitempty"`

		EncryptionPrivateKey *rsa.PrivateKey `json:"encrypt_private_key,omitempty"`
		EncryptionPublicKey  *rsa.PublicKey  `json:"encrypt_public_key,omitempty"`
	}{}

	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	bc.Endpoints = s.Endpoints
	bc.ConnectionPrivateKey = s.ConnectionPrivateKey.toECDSA()
	bc.ConnectionCertificate = s.ConnectionCertificate
	bc.EncryptionPrivateKey = s.EncryptionPrivateKey
	bc.EncryptionPublicKey = s.EncryptionPublicKey

	return nil
}

// PluginConfig plugin configuration
type PluginConfig struct {
	SelfTopicName     string           `json:"self_topic_name"`
	RootTopicName     string           `json:"root_topic_name"`
	RootPublicKey     *rsa.PublicKey   `json:"root_public_key,omitempty"`
	PeersPublicKeys   []*rsa.PublicKey `json:"peers_public_keys,omitempty"`
	PublishQuotaUsage bool             `json:"publish_quota,omitempty"`
}

func (mb *MessageBroker) EncryptionPrivateKey() *rsa.PrivateKey {
	return mb.config.EncryptionPrivateKey
}

func (mb *MessageBroker) GetEndpoints() []string {
	return mb.config.Endpoints
}

func (mb *MessageBroker) EncryptionPublicKey() *rsa.PublicKey {
	return mb.config.EncryptionPublicKey
}

func (mb *MessageBroker) GetKafkaProducer() *kafka.Producer {
	return mb.getProducer()
}

func (mb *MessageBroker) GetKafkaTransactionalProducer() *kafka.Producer {
	return mb.getTransactionalProducer()
}

func (mb *MessageBroker) GetConsumer(consumerGroupID, topicName string, autocommit bool) *kafka.Consumer {
	brokers := strings.Join(mb.config.Endpoints, ",")
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  brokers,
		"group.id":           consumerGroupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": autocommit,
		"isolation.level":    "read_committed",
	})
	if err != nil {
		panic(err)
	}
	err = c.SubscribeTopics([]string{topicName}, nil)
	if err != nil {
		panic(err)
	}

	return c
}

func (mb *MessageBroker) GetRestorationReader(topic string) *kafka.Consumer {
	brokers := strings.Join(mb.config.Endpoints, ",")
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  brokers,
		"auto.offset.reset":  "earliest",
		"group.id":           false,
		"enable.auto.commit": true,
		"isolation.level":    "read_committed",
	})

	if err != nil {
		panic(err)
	}
	err = c.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		panic(err)
	}

	return c
}

func (mb *MessageBroker) SendMessages(msgs []Message, sourceInput *SourceInputMessage) error {
	if len(msgs) == 0 {
		return nil
	}

	if len(msgs) == 1 && sourceInput == nil { // only single message without source
		return mb.sendSingleMessage(msgs[0])
	}

	return mb.sendMessages(msgs, sourceInput)
}

func (mb *MessageBroker) sendMessages(msgs []Message, source *SourceInputMessage) error {
	ctx := context.Background()

	p := mb.GetKafkaTransactionalProducer()
	err := p.BeginTransaction()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		m := mb.prepareMessage(msg)
		err = p.Produce(m, nil)
		if err != nil {
			_ = p.AbortTransaction(ctx)
			return err
		}
	}

	if source != nil {
		// source message offset commit
		err = p.SendOffsetsToTransaction(ctx, source.TopicPartition, source.ConsumerMetadata)
		if err != nil {
			_ = p.AbortTransaction(ctx)
			return err
		}
	}

	err = p.CommitTransaction(ctx)
	if err != nil {
		if err.(kafka.Error).TxnRequiresAbort() {
			_ = p.AbortTransaction(ctx)
			return err
		} else if err.(kafka.Error).IsRetriable() {
			time.Sleep(500 * time.Millisecond)
			return mb.sendMessages(msgs, source) // FIXME: not the best recursive call
		}
		// treat all other errors as fatal errors
		return err
	}

	return nil
}

func (mb *MessageBroker) sendSingleMessage(msg Message) error {
	p := mb.GetKafkaProducer()
	m := mb.prepareMessage(msg)
	err := p.Produce(m, nil)
	if err != nil {
		return fmt.Errorf("producer failed: %s - %s: %v: %+v", err.Error(), p.String(), p, msg)
	}

	e := <-p.Events()
	switch ev := e.(type) { // nolint: gocritic
	case *kafka.Message:
		if ev.TopicPartition.Error != nil {
			return fmt.Errorf("delivery failed: %s", e.String())
		}
		return nil
	}

	return nil
}

func (mb *MessageBroker) prepareMessage(msg Message) *kafka.Message {
	km := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Partition: kafka.PartitionAny,
			Topic:     &msg.Topic,
		},
		Value:         msg.Value,
		Key:           []byte(msg.Key),
		TimestampType: kafka.TimestampCreateTime,
	}
	if len(msg.Headers) > 0 {
		headers := make([]kafka.Header, 0)
		for k, v := range msg.Headers {
			headers = append(headers, kafka.Header{Key: k, Value: v})
		}
		km.Headers = headers
	}

	return km
}

func (mb *MessageBroker) getProducer() *kafka.Producer {
	mb.producerSync.Do(func() {
		brokers := strings.Join(mb.config.Endpoints, ",")
		p, err := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": brokers,
			// "batch.size": 16384,
			"client.id": mb.PluginConfig.SelfTopicName,
		})
		if err != nil {
			panic(err)
		}
		mb.producer = p
	})

	return mb.producer
}

func (mb *MessageBroker) getTransactionalProducer() *kafka.Producer {
	mb.transProducerSync.Do(func() {
		brokers := strings.Join(mb.config.Endpoints, ",")
		p, err := kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": brokers,
			"transactional.id":  mb.PluginConfig.SelfTopicName,
			// "batch.size": 16384,
			"client.id": mb.PluginConfig.SelfTopicName,
		})
		if err != nil {
			panic(err)
		}
		err = p.InitTransactions(context.TODO())
		if err != nil {
			panic(err)
		}
		mb.transProducer = p
	})

	return mb.transProducer
}

func (mb *MessageBroker) CreateTopic(ctx context.Context, topic string) error {
	brokers := strings.Join(mb.config.Endpoints, ",")
	ac, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return err
	}

	repFactor := 1
	inSyncReplicas := 1
	if len(mb.config.Endpoints) > 1 {
		repFactor = len(mb.config.Endpoints)
		inSyncReplicas = len(mb.config.Endpoints) - 1
	}

	tc := kafka.TopicSpecification{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: repFactor,
		Config:            map[string]string{"min.insync.replicas": strconv.FormatInt(int64(inSyncReplicas), 10)},
	}
	res, err := ac.CreateTopics(ctx, []kafka.TopicSpecification{tc})
	if err != nil {
		return err
	}
	if res[0].Error.Error() != "" {
		switch res[0].Error.Code() {
		case kafka.ErrNoError, kafka.ErrTopicAlreadyExists:
			return nil

		default:
			return res[0].Error
		}
	}

	return nil
}

func (mb *MessageBroker) DeleteTopic(ctx context.Context, topicName string) error {
	brokers := strings.Join(mb.config.Endpoints, ",")
	ac, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
	if err != nil {
		return err
	}
	res, err := ac.DeleteTopics(ctx, []string{topicName})
	if err != nil {
		return err
	}
	if res[0].Error.Error() != "" {
		return res[0].Error
	}

	return nil
}
