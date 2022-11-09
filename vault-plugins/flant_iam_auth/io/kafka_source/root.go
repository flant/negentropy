package kafka_source

import (
	"crypto/sha256"

	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/root"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func NewRootKafkaSource(kf *sharedkafka.MessageBroker, modelsHandler root.ModelHandler, parentLogger hclog.Logger) *io.KafkaSourceImpl {
	runConsumerGroupIDProvider := func(kf *sharedkafka.MessageBroker) string {
		return kf.PluginConfig.SelfTopicName
	}
	topicNameProvider := func(kf *sharedkafka.MessageBroker) string {
		return kf.PluginConfig.RootTopicName
	}
	verifySign := func(signature []byte, messageValue []byte) error {
		hashed := sha256.Sum256(messageValue)
		return sharedkafka.VerifySignature(signature, kf.PluginConfig.RootPublicKey, hashed)
	}
	decrypt := func(encryptedMessageValue []byte, chunked bool) ([]byte, error) {
		return sharedkafka.NewEncrypter().Decrypt(encryptedMessageValue, kf.EncryptionPrivateKey(), chunked)
	}
	processRunMessage := func(txn io.Txn, msg sharedkafka.MsgDecoded) error {
		return root.HandleNewMessageIamRootSource(txn, modelsHandler, msg)
	}

	return &io.KafkaSourceImpl{
		NameOfSource:              "authRootKafkaSource",
		KafkaBroker:               kf,
		Logger:                    parentLogger.Named("authRootKafkaSource"),
		ProvideRunConsumerGroupID: runConsumerGroupIDProvider,
		ProvideTopicName:          topicNameProvider,
		VerifySign:                verifySign,
		Decrypt:                   decrypt,
		ProcessRunMessage:         processRunMessage,
		ProcessRestoreMessage:     root.HandleRestoreMessagesRootSource,
		Runnable:                  true,
	}
}
