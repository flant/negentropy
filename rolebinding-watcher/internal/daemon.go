package internal

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/rolebinding-watcher/pkg"
	ext_ff_io "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/io"
	ext_ff_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	ext_sa_io "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/io"
	ext_sa_repo "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_source"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type Daemon struct {
	memstorage *sharedio.MemoryStore
	logger     hclog.Logger
}

func (d *Daemon) Run(userEffectiveRoleProcessor UserEffectiveRoleProcessor) error {
	hooker := &Hooker{
		Logger:    d.logger.Named("hooker"),
		processor: &ChangesProcessor{userEffectiveRoleProcessor: userEffectiveRoleProcessor, Logger: d.logger},
	}
	hooker.RegisterHooks(d.memstorage)
	d.memstorage.RunKafkaSourceMainLoops()
	time.Sleep(time.Hour * 1000000) // TODO remake it
	return nil
}

func NewDaemon(kafkaCFG sharedkafka.BrokerConfig, topicName string,
	runConsumerID string, parentLogger hclog.Logger) (*Daemon, error) {
	mb := messageBroker(kafkaCFG, runConsumerID, parentLogger)
	storage, err := memStorage(mb, parentLogger)
	if err != nil {
		return nil, err
	}

	storage.AddKafkaSource(kafkaSource(mb, topicName, runConsumerID, parentLogger))
	return &Daemon{memstorage: storage, logger: parentLogger.Named("daemon")}, nil
}

func messageBroker(kafkaCFG sharedkafka.BrokerConfig, consumerGroupID string, parentLogger hclog.Logger) *sharedkafka.MessageBroker {
	fakePluginCfg := sharedkafka.PluginConfig{
		SelfTopicName: consumerGroupID, // need for valid committing reading
	}

	mb := &sharedkafka.MessageBroker{
		Logger:       parentLogger.Named("mb"),
		KafkaConfig:  kafkaCFG,
		PluginConfig: fakePluginCfg,
	}
	mb.CheckConfig()
	return mb
}

func memStorage(mb *sharedkafka.MessageBroker, parentLogger hclog.Logger) (*sharedio.MemoryStore, error) {
	iamSchema, err := iam_repo.GetSchema()
	if err != nil {
		return nil, err
	}
	ffSchema, err := ext_ff_repo.GetSchema()
	if err != nil {
		return nil, err
	}

	schema, err := memdb.MergeDBSchemasAndValidate(iamSchema, ext_sa_repo.ServerSchema(), ffSchema,
		pkg.UserEffectiveRolesSchema())
	if err != nil {
		return nil, err
	}
	// copy of data from iam, so no needs to checks
	schema = memdb.DropRelations(schema)

	return sharedio.NewMemoryStore(schema, mb, parentLogger)
}

func msgHandler(txn sharedio.Txn, msg sharedio.MsgDecoded) (err error) {
	for _, r := range []kafka_source.RestoreFunc{
		jwtkafka.SelfRestoreMessage,
		ext_sa_io.HandleServerAccessObjects,
		ext_ff_io.HandleFlantFlowObjects,
		kafka_source.IamObjectsRestoreHandler,
	} {
		handled, err := r(txn, msg)
		if err != nil {
			return err
		}

		if handled {
			return nil
		}
	}
	return fmt.Errorf("type= %s: %w", msg.Type, consts.ErrNotHandledObject)
}

func kafkaSource(mb *sharedkafka.MessageBroker, topicName string,
	runConsumerID string, parentLogger hclog.Logger) *sharedio.KafkaSourceImpl {
	return &sharedio.KafkaSourceImpl{
		NameOfSource: "rolebinding-watcher-consumer",
		KafkaBroker:  mb,
		Logger:       parentLogger.Named("kafka-consumer"),
		ProvideRunConsumerGroupID: func(kf *sharedkafka.MessageBroker) string {
			return runConsumerID
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
		ProcessRunMessage: func(txn sharedio.Txn, msg sharedio.MsgDecoded) error {
			parentLogger.Debug("message", "key", msg.Key())
			return msgHandler(txn, msg)
		},
		ProcessRestoreMessage:        msgHandler,
		IgnoreSourceInputMessageBody: true,
		Runnable:                     true,
	}
}
