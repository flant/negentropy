package kafka

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// TopicType represents kafka topic type
type TopicType string

func (t TopicType) String() string {
	return string(t)
}

type MessageBroker struct {
	isConfigured bool

	producerSync sync.Once
	producer     *kafka.Writer

	config        BrokerConfig
	selfHealTopic string
}

func (mb *MessageBroker) Configured() bool {
	return mb.isConfigured
}

type BrokerConfig struct {
	Endpoints []string

	ConnectionPrivateKey  *ecdsa.PrivateKey
	ConnectionCertificate *x509.Certificate

	EncryptionPrivateKey *rsa.PrivateKey
	EncryptionPublicKey  *rsa.PublicKey
}

func (mb *MessageBroker) EncryptionPrivateKey() *rsa.PrivateKey {
	return mb.config.EncryptionPrivateKey
}

func (mb *MessageBroker) EncryptionPublicKey() *rsa.PublicKey {
	return mb.config.EncryptionPublicKey
}

func (mb *MessageBroker) GetKafkaWriter() *kafka.Writer {
	return mb.getProducer()
}

func (mb *MessageBroker) GetKafkaReader(replicaName, topicName string) *kafka.Reader {
	// TODO: tls
	rd := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        mb.config.Endpoints,
		GroupID:        "replica." + replicaName,
		GroupTopics:    nil,
		Topic:          topicName,
		MinBytes:       0,
		MaxBytes:       1048576 * 4, // 4Mb
		MaxWait:        15 * time.Second,
		IsolationLevel: kafka.ReadCommitted, // must have
	})

	return rd
}

func (mb *MessageBroker) GetRestorationReader(replicaName, topicName string) *kafka.Reader {
	if replicaName != "" {
		topicName = topicName + "." + replicaName
	}
	// TODO: tls
	rd := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        mb.config.Endpoints,
		Topic:          topicName,
		MinBytes:       1,
		MaxBytes:       1048576 * 4, // 4Mb
		MaxWait:        3 * time.Second,
		IsolationLevel: kafka.ReadCommitted,
	})

	return rd
}

func (mb *MessageBroker) GetLastOffset(topic string) (int64, error) {
	return getPartitionsForTopic(mb.config.Endpoints[0], topic)
}

func getPartitionsForTopic(endpoint string, topic string) (int64, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", endpoint, topic, 0)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	return conn.ReadLastOffset()
}

func (mb *MessageBroker) getProducer() *kafka.Writer {
	mb.producerSync.Do(func() {
		w := &kafka.Writer{
			Addr:         kafka.TCP(mb.config.Endpoints...),
			MaxAttempts:  10,
			RequiredAcks: kafka.RequireAll,
			Async:        false,
			BatchSize:    300,
			BatchBytes:   1048576 * 4, // 4Mb
			Transport:    nil,         // TODO: transport
		}

		mb.producer = w
	})

	return mb.producer
}

func (mb *MessageBroker) CreateTopic(topics ...kafka.TopicConfig) error {
	repFactor := 1
	inSyncReplicas := 1
	if len(mb.config.Endpoints) > 1 {
		repFactor = len(mb.config.Endpoints)
		inSyncReplicas = len(mb.config.Endpoints) - 1
	}

	for i, topic := range topics {
		topic.ReplicationFactor = repFactor
		insyncConf := kafka.ConfigEntry{
			ConfigName:  "min.insync.replicas",
			ConfigValue: strconv.FormatInt(int64(inSyncReplicas), 10),
		}

		topic.ConfigEntries = append(topic.ConfigEntries, insyncConf)
		if topic.NumPartitions == 0 {
			topic.NumPartitions = 1
		}

		topics[i] = topic
	}

	return mb.createTopicsWithFallback(topics)
}

func (mb *MessageBroker) DeleteTopic(topicName ...string) error {
	return mb.deleteTopicsWithFallback(topicName)
}

func (mb *MessageBroker) createTopicsWithFallback(topics []kafka.TopicConfig) error {
	var err error
	for _, broker := range mb.config.Endpoints {
		err = mb.createTopics(broker, topics)
		if err == nil {
			return nil
		}
	}

	return err
}

func (mb *MessageBroker) deleteTopicsWithFallback(topics []string) error {
	var err error
	for _, broker := range mb.config.Endpoints {
		err = mb.deleteTopics(broker, topics)
		if err == nil {
			return nil
		}
	}

	return err
}

func (mb *MessageBroker) createTopics(endpoint string, topicConfigs []kafka.TopicConfig) error {
	conn, err := kafka.Dial("tcp", endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	return controllerConn.CreateTopics(topicConfigs...)
}

func (mb *MessageBroker) deleteTopics(endpoint string, topics []string) error {
	conn, err := kafka.Dial("tcp", endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	return controllerConn.DeleteTopics(topics...)
}
