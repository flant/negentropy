package internal

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	log "github.com/hashicorp/go-hclog"

	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

type DecryptedMessageProceeder interface {
	ProceedMessage(msgBody []byte, msgKey []byte) error
}

type NegentropyKafkaSource struct {
	kf                        *sharedkafka.MessageBroker
	decryptor                 *sharedkafka.Encrypter
	topicName                 string
	groupID                   string
	stopC                     chan struct{}
	decryptedMessageProceeder DecryptedMessageProceeder
	logger                    log.Logger
}

func NewKafkaSource(kafkaConfig sharedkafka.BrokerConfig, topicName, groupID string,
	parentLogger log.Logger, proceeder DecryptedMessageProceeder) (*NegentropyKafkaSource, error) {
	mb := &sharedkafka.MessageBroker{
		Logger:      parentLogger.Named("mb"),
		KafkaConfig: kafkaConfig,
	}

	return &NegentropyKafkaSource{
		kf:                        mb,
		decryptor:                 sharedkafka.NewEncrypter(),
		logger:                    parentLogger.Named("negentropyKafkaSource"),
		topicName:                 topicName,
		groupID:                   groupID,
		stopC:                     make(chan struct{}),
		decryptedMessageProceeder: proceeder,
	}, nil
}

func (mks *NegentropyKafkaSource) msgHandler(sourceConsumer *kafka.Consumer, msg *kafka.Message) {
	logger := mks.logger.Named("msgHandler")
	logger.Trace("started")
	defer logger.Trace("exit")
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		logger.Error(fmt.Sprintf("key has wong format: %s", string(msg.Key)))
		return
	}

	var signature []byte
	var chunked bool
	for _, header := range msg.Headers {
		switch header.Key {
		case "signature":
			signature = header.Value

		case "chunked":
			chunked = true
		}
	}

	decrypted, err := mks.decryptor.Decrypt(msg.Value, mks.kf.EncryptionPrivateKey(), chunked)
	if err != nil {
		logger.Error(fmt.Sprintf("err: %s", err.Error()))
		return
	}
	hashed := sha256.Sum256(decrypted)
	err = sharedkafka.VerifySignature(signature, mks.kf.EncryptionPublicKey(), hashed)
	if err != nil {
		logger.Error(fmt.Sprintf("err: %s", err.Error()))
		return
	}

	err = mks.decryptedMessageProceeder.ProceedMessage(msg.Key, decrypted)
	if err != nil {
		logger.Error(fmt.Sprintf("key: %s err: %s", string(msg.Key), string(err.Error())))
		return
	} else {
		logger.Debug("success", "key", string(msg.Key))
	}

	_, err = sourceConsumer.CommitMessage(msg)
	if err != nil {
		logger.Error(fmt.Sprintf("err: %s", err.Error()))
		return
	}
	logger.Trace("normal finish")
}

func (mks *NegentropyKafkaSource) Run() {
	mks.logger.Debug("Watcher - start", "topic", mks.topicName, "groupID", mks.groupID)
	defer mks.logger.Debug("Watcher - stop", "topic", mks.topicName, "groupID", mks.groupID)
	runConsumer, err := mks.kf.GetSubscribedRunConsumer(mks.groupID, mks.topicName)
	if err != nil {
		panic(err) // it is critical error for application which can be crashed
	}
	defer sharedkafka.DeferredСlose(runConsumer, mks.logger)
	io.RunMessageLoop(runConsumer, mks.msgHandler, mks.stopC, mks.logger)
}

func (mks *NegentropyKafkaSource) Stop() {
	mks.stopC <- struct{}{}
}

func ParseRSAPubKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	pksTrimmed := strings.ReplaceAll(strings.TrimSpace(publicKeyPEM), "\\n", "\n")
	pub, err := utils.ParsePubkey(pksTrimmed)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

func ParseRSAPrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	data := strings.ReplaceAll(strings.TrimSpace(privateKeyPEM), "\\n", "\n")
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("private key can not be parsed")
	}
	prvkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return prvkey, nil
}
