package pkg

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

type DecryptedMessageProceeder interface {
	ProceedMessage(msgBody []byte, msgKey []byte) error
}

type KafkaSource struct {
	io.KafkaSourceImpl
	// need for using KafkaSourceImpl
	io.MemoryStore
}

func (k *KafkaSource) Run() {
	k.KafkaSourceImpl.Run(&k.MemoryStore)
}

func (k *KafkaSource) Stop() {
	k.KafkaSourceImpl.Stop()
}

func NewKafkaSource(kafkaCFG sharedkafka.BrokerConfig, topicName, groupID string, parentLogger hclog.Logger, proceeder DecryptedMessageProceeder) *KafkaSource {
	fakePluginCfg := sharedkafka.PluginConfig{
		SelfTopicName: groupID, // need for valid committing reading
	}

	mb := &sharedkafka.MessageBroker{
		Logger:       parentLogger.Named("mb"),
		KafkaConfig:  kafkaCFG,
		PluginConfig: fakePluginCfg,
	}

	ks := &io.KafkaSourceImpl{
		NameOfSource: "outer-kafka-consumer",
		KafkaBroker:  mb,
		Logger:       parentLogger.Named("kafka-consumer"),
		ProvideRunConsumerGroupID: func(kf *sharedkafka.MessageBroker) string {
			return groupID
		},
		ProvideTopicName: func(kf *sharedkafka.MessageBroker) string {
			return topicName
		},
		VerifySign: func(signature []byte, messageValue []byte) error {
			hashed := sha256.Sum256(messageValue)
			return sharedkafka.VerifySignature(signature, mb.EncryptionPublicKey(), hashed)
		},
		Decrypt: func(encryptedMessageValue []byte, chunked bool) ([]byte, error) {
			return sharedkafka.NewEncrypter().Decrypt(encryptedMessageValue, mb.EncryptionPrivateKey(), chunked)
		},
		ProcessRunMessage: func(_ io.Txn, msg io.MsgDecoded) error {
			return proceeder.ProceedMessage([]byte(msg.Key()), msg.Data)
		},
		IgnoreSourceInputMessageBody: true,
		Runnable:                     true,
	}

	return &KafkaSource{
		KafkaSourceImpl: *ks,
		MemoryStore:     EmptyMemstore(mb, parentLogger.Named("memstore")),
	}
}

func EmptyMemstore(kb *sharedkafka.MessageBroker, logger hclog.Logger) io.MemoryStore {
	ms, err := io.NewMemoryStore(
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
		}, kb, logger,
	)
	if err != nil {
		panic(err)
	}
	return *ms
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
