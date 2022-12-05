package kafka

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/utils"
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

	// IgnoreBody send only offset, ignoring body payload
	IgnoreBody bool
}

var emptyTopicPartition = kafka.TopicPartition{}

func NewSourceInputMessage(c *kafka.Consumer, tp kafka.TopicPartition) (*SourceInputMessage, error) {
	if tp == emptyTopicPartition {
		return nil, fmt.Errorf("empty topic partition")
	}
	cm, err := c.GetConsumerGroupMetadata()
	if err != nil {
		return nil, err
	}
	tp.Offset++ // need commit next offset
	return &SourceInputMessage{
		TopicPartition:   []kafka.TopicPartition{tp},
		ConsumerMetadata: cm,
	}, nil
}

type MessageBroker struct {
	isConfigured bool

	producerSync      sync.Once
	producer          *kafka.Producer
	transProducerSync sync.Once
	transProducer     *kafka.Producer

	KafkaConfig  BrokerConfig
	PluginConfig PluginConfig

	Logger log.Logger
}

func NewMessageBroker(ctx context.Context, storage logical.Storage, parentLogger log.Logger) (*MessageBroker, error) {
	mb := &MessageBroker{
		Logger: parentLogger.Named("MessageBroker"),
	}
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

		mb.KafkaConfig = config
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
	if len(mb.KafkaConfig.Endpoints) > 0 &&
		mb.KafkaConfig.EncryptionPublicKey != nil &&
		mb.KafkaConfig.EncryptionPrivateKey != nil &&
		mb.PluginConfig.SelfTopicName != "" {
		mb.isConfigured = true
	}
}

func (mb *MessageBroker) Configured() bool {
	return mb.isConfigured
}

func (mb *MessageBroker) withSSLConfig(config *kafka.ConfigMap) *kafka.ConfigMap {
	return mb.KafkaConfig.SSLConfig.addSSLConfig(config)
}

func (mb *MessageBroker) CheckConnection(endpoints []string) error {
	brokers := strings.Join(endpoints, ",")
	ac, err := kafka.NewAdminClient(mb.withSSLConfig(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"retries":           3,
	}))
	if err != nil {
		return err
	}
	defer ac.Close()

	_, err = ac.ClusterID(context.TODO())
	return err
}

// SSLConfig configures ssl connection to kafka
type SSLConfig struct {
	UseSSL                bool   `json:"use_ssl,omitempty"`
	CAPath                string `json:"ca_path,omitempty"`                 //  "ssl.ca.location":          /etc/ca.crt
	ClientPrivateKeyPath  string `json:"client_private_key_path,omitempty"` //  "ssl.key.location":         /etc/client.key
	ClientCertificatePath string `json:"client_certificate_path,omitempty"` //  "ssl.certificate.location": /etc/client.crt
}

func (cfg *SSLConfig) addSSLConfig(config *kafka.ConfigMap) *kafka.ConfigMap {
	if cfg != nil && cfg.UseSSL {
		(*config)["security.protocol"] = "ssl"
		(*config)["ssl.ca.location"] = cfg.CAPath
		(*config)["ssl.key.location"] = cfg.ClientPrivateKeyPath
		(*config)["ssl.certificate.location"] = cfg.ClientCertificatePath
	}
	return config
}

// BrokerConfig connection config
type BrokerConfig struct {
	Endpoints []string `json:"endpoints"`

	SSLConfig *SSLConfig `json:"ssl_config,omitempty"`

	// Self pair of keys from this vault plugin instance (or plugin privateKey and root publicKey for root source reading)
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

		SSLConfig *SSLConfig `json:"ssl_config,omitempty"`

		EncryptionPrivateKey *rsa.PrivateKey `json:"encrypt_private_key,omitempty"`
		EncryptionPublicKey  *rsa.PublicKey  `json:"encrypt_public_key,omitempty"`
	}{}

	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	bc.SSLConfig = s.SSLConfig
	bc.Endpoints = s.Endpoints
	bc.EncryptionPrivateKey = s.EncryptionPrivateKey
	bc.EncryptionPublicKey = s.EncryptionPublicKey

	return nil
}

// PluginConfig plugin configuration
type PluginConfig struct {
	SelfTopicName string `json:"self_topic_name"`
	RootTopicName string `json:"root_topic_name"`
	// RootPublicKey public rsa key from Root Source vault
	RootPublicKey     *rsa.PublicKey   `json:"root_public_key,omitempty"`
	PeersPublicKeys   []*rsa.PublicKey `json:"peers_public_keys,omitempty"`
	PublishQuotaUsage bool             `json:"publish_quota,omitempty"`
}

func (mb *MessageBroker) EncryptionPrivateKey() *rsa.PrivateKey {
	return mb.KafkaConfig.EncryptionPrivateKey
}

func (mb *MessageBroker) GetEndpoints() []string {
	return mb.KafkaConfig.Endpoints
}

func (mb *MessageBroker) EncryptionPublicKey() *rsa.PublicKey {
	return mb.KafkaConfig.EncryptionPublicKey
}

func (mb *MessageBroker) GetEncryptionPublicKeyStrict() (string, error) {
	k := mb.KafkaConfig.EncryptionPublicKey
	if k == nil {
		return "", fmt.Errorf("cannot getting kafka public key. may be kafka is not configure")
	}

	return utils.DecodePemKey(k), nil
}

// func (mb *MessageBroker) GetKafkaProducer() (*kafka.Producer, error) {
//	return mb.getProducer()
// }

func (mb *MessageBroker) GetKafkaTransactionalProducer() (*kafka.Producer, error) {
	return mb.getTransactionalProducer()
}

func (mb *MessageBroker) getUnsubscribedConsumer(consumerGroupID string) (*kafka.Consumer, error) {
	brokers := strings.Join(mb.KafkaConfig.Endpoints, ",")
	c, err := kafka.NewConsumer(mb.withSSLConfig(&kafka.ConfigMap{
		"bootstrap.servers":        brokers,
		"group.id":                 consumerGroupID,
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       false,
		"isolation.level":          "read_committed",
		"go.events.channel.enable": true,
	}))
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (mb *MessageBroker) GetUnsubscribedRunConsumer(consumerGroupID string) (*kafka.Consumer, error) {
	return mb.getUnsubscribedConsumer(consumerGroupID)
}

func (mb *MessageBroker) GetSubscribedRunConsumer(consumerGroupID, topicName string) (*kafka.Consumer, error) {
	c, err := mb.GetUnsubscribedRunConsumer(consumerGroupID)
	if err != nil {
		return nil, err
	}
	err = c.Subscribe(topicName, nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetRestorationReader returns Unsubscribed for any topic consumer
func (mb *MessageBroker) GetRestorationReader() (*kafka.Consumer, error) {
	return mb.getUnsubscribedConsumer(fmt.Sprintf("restoration_reader_%d", time.Now().Unix()))
}

func (mb *MessageBroker) SendMessages(msgs []Message, sourceInput *SourceInputMessage) error {
	mb.Logger.Debug("SendMessages - started")
	defer mb.Logger.Debug("SendMessages - exit")
	if sourceInput != nil && (sourceInput.IgnoreBody || len(msgs) == 0) {
		return mb.sendOffset(sourceInput)
	}

	if len(msgs) == 0 {
		return nil
	}

	return mb.sendMessages(msgs, sourceInput)
}

func (mb *MessageBroker) sendOffset(sourceInput *SourceInputMessage) error {
	if sourceInput == nil {
		return nil
	}

	ctx := context.Background()

	p, err := mb.GetKafkaTransactionalProducer()
	if err != nil {
		return err
	}
	err = p.BeginTransaction()
	if err != nil {
		return err
	}

	err = p.SendOffsetsToTransaction(ctx, sourceInput.TopicPartition, sourceInput.ConsumerMetadata)
	if err != nil {
		_ = p.AbortTransaction(ctx)
		return err
	}

	return p.CommitTransaction(ctx)
}

func (mb *MessageBroker) sendMessages(msgs []Message, source *SourceInputMessage) error {
	ctx := context.Background()

	p, err := mb.GetKafkaTransactionalProducer()
	if err != nil {
		return err
	}
	err = p.BeginTransaction()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		m := mb.prepareMessage(msg)
		err = p.Produce(m, nil)
		if err != nil {
			err2 := p.AbortTransaction(ctx)
			if err2 != nil {
				mb.Logger.Error("error aborting transaction: %w", err2)
			}
			return err
		}
	}

	if source != nil {
		// source message offset commit
		err = p.SendOffsetsToTransaction(ctx, source.TopicPartition, source.ConsumerMetadata)
		if err != nil {
			err2 := p.AbortTransaction(ctx)
			if err2 != nil {
				mb.Logger.Error("error aborting transaction: %w", err2)
			}
			return err
		}
	}

	err = p.CommitTransaction(ctx)

	if err != nil {
		if err.(kafka.Error).TxnRequiresAbort() {
			err2 := p.AbortTransaction(ctx)
			if err2 != nil {
				mb.Logger.Error("error aborting transaction: %w", err2)
			}
			return err
		} else if err.(kafka.Error).IsRetriable() {
			mb.Logger.Info(fmt.Sprintf("got err.(kafka.Error).IsRetriable():%s, retry in 5 seconds", err.Error()))
			time.Sleep(500 * time.Millisecond)
			return mb.sendMessages(msgs, source) // FIXME: not the best recursive call
		}
		// treat all other errors as fatal errors
		return err
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

func (mb *MessageBroker) getProducer() (*kafka.Producer, error) {
	var err error = nil
	var p *kafka.Producer
	mb.producerSync.Do(func() {
		brokers := strings.Join(mb.KafkaConfig.Endpoints, ",")
		p, err = kafka.NewProducer(mb.withSSLConfig(&kafka.ConfigMap{
			"bootstrap.servers": brokers,
			// "batch.size": 16384,
			"client.id": mb.PluginConfig.SelfTopicName + ".producer",
		}))
		if err != nil {
			return
		}
		mb.producer = p
	})

	return mb.producer, err
}

func (mb *MessageBroker) getTransactionalProducer() (*kafka.Producer, error) {
	var err error
	var p *kafka.Producer
	mb.transProducerSync.Do(func() {
		brokers := strings.Join(mb.KafkaConfig.Endpoints, ",")
		p, err = kafka.NewProducer(mb.withSSLConfig(&kafka.ConfigMap{
			"bootstrap.servers": brokers,
			"transactional.id":  mb.PluginConfig.SelfTopicName,
			// "batch.size": 16384,
			"client.id": mb.PluginConfig.SelfTopicName + ".transactional_producer",
		}))
		if err != nil {
			return
		}
		err = p.InitTransactions(context.TODO())
		if err != nil {
			return
		}
		mb.transProducer = p
	})

	return mb.transProducer, nil
}

func (mb *MessageBroker) CreateTopic(ctx context.Context, topic string, config map[string]string) error {
	brokers := strings.Join(mb.KafkaConfig.Endpoints, ",")
	ac, err := kafka.NewAdminClient(mb.withSSLConfig(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	}))
	if err != nil {
		return err
	}

	repFactor := 1
	inSyncReplicas := 1
	if len(mb.KafkaConfig.Endpoints) > 1 {
		repFactor = len(mb.KafkaConfig.Endpoints)
		inSyncReplicas = len(mb.KafkaConfig.Endpoints) - 1
	}

	tc := kafka.TopicSpecification{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: repFactor,
		Config: map[string]string{
			"min.insync.replicas": strconv.FormatInt(int64(inSyncReplicas), 10),
			"cleanup.policy":      "compact",
		},
	}

	for k, v := range config {
		tc.Config[k] = v
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
	ac.Close()
	return nil
}

func (mb *MessageBroker) DeleteTopic(ctx context.Context, topicName string) error {
	brokers := strings.Join(mb.KafkaConfig.Endpoints, ",")
	ac, err := kafka.NewAdminClient(mb.withSSLConfig(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	}))
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

func (mb *MessageBroker) Close() {
	if mb.producer != nil {
		mb.producer.Close()
		mb.producer = nil
	}
	if mb.transProducer != nil {
		mb.transProducer.Close()
		mb.transProducer = nil
	}
}

// TopicExists
func (mb *MessageBroker) TopicExists(topic string) (bool, error) {
	groupID := fmt.Sprintf("temp_group_%d", time.Now().Unix())
	consumer, err := mb.getUnsubscribedConsumer(groupID)
	if err != nil {
		return false, err
	}
	defer DeferredСlose(consumer, mb.Logger)

	var meta *kafka.Metadata = nil

	for meta == nil {
		meta, err = consumer.GetMetadata(&topic, false, 10000)
		if kErr, ok := err.(kafka.Error); ok && kErr.Code() == kafka.ErrTransport {
			err = nil
			meta = nil
			time.Sleep(time.Millisecond * 100)

		}
		if err != nil {
			return false, err
		}
	}
	if topicMeta := meta.Topics[topic]; topicMeta.Error.Code() == kafka.ErrUnknownTopicOrPart {
		return false, nil
	}
	return true, nil
}

type Closable interface {
	Close() error
}

// DeferredСlose closes closable at separate goroutine
func DeferredСlose(closable Closable, logger log.Logger) {
	go func() {
		err := closable.Close()
		if err != nil {
			logger.Warn(fmt.Sprintf("error during closing %#v: %s", closable, err.Error()))
		}
	}()
}
