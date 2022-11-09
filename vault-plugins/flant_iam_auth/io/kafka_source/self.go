package kafka_source

import (
	"crypto/sha256"

	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/kafka_handlers/self"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	jwtkafka "github.com/flant/negentropy/vault-plugins/shared/jwt/kafka"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func NewSelfKafkaSource(kf *sharedkafka.MessageBroker, handler self.ModelHandler, parentLogger hclog.Logger) *io.KafkaSourceImpl {
	runConsumerGroupIDProvider := func(kf *sharedkafka.MessageBroker) string {
		return kf.PluginConfig.SelfTopicName
	}
	topicNameProvider := func(kf *sharedkafka.MessageBroker) string {
		return kf.PluginConfig.SelfTopicName
	}
	verifySign := func(signature []byte, messageValue []byte) error {
		hashed := sha256.Sum256(messageValue)
		return sharedkafka.VerifySignature(signature, kf.EncryptionPublicKey(), hashed)
	}
	decrypt := func(encryptedMessageValue []byte, chunked bool) ([]byte, error) {
		return sharedkafka.NewEncrypter().Decrypt(encryptedMessageValue, kf.EncryptionPrivateKey(), chunked)
	}
	processRunMessage := func(txn io.Txn, msg sharedkafka.MsgDecoded) error {
		return self.HandleNewMessageSelfSource(txn, handler, &msg)
	}
	processRestoreMessage := func(txn io.Txn, msg sharedkafka.MsgDecoded) error {
		return self.HandleRestoreMessagesSelfSource(txn, msg, []self.RestoreFunc{
			jwtkafka.SelfRestoreMessage,
		})
	}

	return &io.KafkaSourceImpl{
		NameOfSource:              "authSelfKafkaSource",
		KafkaBroker:               kf,
		Logger:                    parentLogger.Named("authSelfKafkaSource"),
		ProvideRunConsumerGroupID: runConsumerGroupIDProvider,
		ProvideTopicName:          topicNameProvider,
		VerifySign:                verifySign,
		Decrypt:                   decrypt,
		ProcessRunMessage:         processRunMessage,
		ProcessRestoreMessage:     processRestoreMessage,
		Runnable:                  true,
	}
}
