package pkg

import (
	"github.com/hashicorp/go-hclog"
	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type DecryptedMessageProceeder interface {
	ProceedMessage(msg io.MsgDecoded) error
}

type KafkaSource struct {
	io.KafkaSourceImpl
	// need for using KafkaSourceImpl
	*io.MemoryStore
}

func (k *KafkaSource) Run() {
	k.KafkaSourceImpl.Run(k.MemoryStore)
}

func (k *KafkaSource) Stop() {
	k.KafkaSourceImpl.Stop()
}

func NewKafkaSource(messageBroker *sharedkafka.MessageBroker, memStore *io.MemoryStore, topicName, groupID string, parentLogger hclog.Logger,
	proceeder DecryptedMessageProceeder) *KafkaSource {

	ks := &io.KafkaSourceImpl{
		NameOfSource: "item-watcher-" + topicName,
		KafkaBroker:  messageBroker,
		Logger:       parentLogger.Named("item-watcher"),
		ProvideRunConsumerGroupID: func(kf *sharedkafka.MessageBroker) string {
			return groupID
		},
		ProvideTopicName: func(kf *sharedkafka.MessageBroker) string {
			return topicName
		},
		VerifySign: nil,
		Decrypt:    nil,
		ProcessRunMessage: func(_ io.Txn, msg io.MsgDecoded) error {
			return proceeder.ProceedMessage(msg)
		},
		IgnoreSourceInputMessageBody: true,
		Runnable:                     true,
	}

	return &KafkaSource{
		KafkaSourceImpl: *ks,
		MemoryStore:     memStore,
	}
}

func MessageBroker(kafkaCFG sharedkafka.BrokerConfig, groupID string, parentLogger hclog.Logger) *sharedkafka.MessageBroker {
	fakePluginCfg := sharedkafka.PluginConfig{
		SelfTopicName: groupID, // need for valid committing reading
	}

	mb := &sharedkafka.MessageBroker{
		Logger:       parentLogger.Named("mb"),
		KafkaConfig:  kafkaCFG,
		PluginConfig: fakePluginCfg,
	}
	return mb
}

func EmptyMemstore(kb *sharedkafka.MessageBroker, logger hclog.Logger) *io.MemoryStore {
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
	return ms
}
