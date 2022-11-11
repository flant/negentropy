package kafka

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const topicName = "jwks"

func NewJWKSKafkaSource(kf *sharedkafka.MessageBroker, parentLogger hclog.Logger) *io.KafkaSourceImpl {
	runConsumerGroupIDProvider := func(kf *sharedkafka.MessageBroker) string {
		return kf.PluginConfig.SelfTopicName
	}
	topicNameProvider := func(_ *sharedkafka.MessageBroker) string {
		return topicName
	}
	verifySign := func(signature []byte, messageValue []byte) error {
		hashed := sha256.Sum256(messageValue)

		for _, pub := range kf.PluginConfig.PeersPublicKeys {
			err := sharedkafka.VerifySignature(signature, pub, hashed)
			if err == nil {
				return nil
			}
		}

		return fmt.Errorf("no public key for signature found")
	}

	return &io.KafkaSourceImpl{
		NameOfSource:                    topicName,
		KafkaBroker:                     kf,
		Logger:                          parentLogger.Named("jwksKafkaSource"),
		ProvideRunConsumerGroupID:       runConsumerGroupIDProvider,
		ProvideTopicName:                topicNameProvider,
		VerifySign:                      verifySign,
		Decrypt:                         nil, // this common topic is not encrypted
		ProcessRunMessage:               processMessage,
		ProcessRestoreMessage:           processMessage,
		IgnoreSourceInputMessageBody:    true, // this topic has unusual Commit mechanic
		SkipRestorationOnWrongSignature: true, // this topic has unusual content
		Runnable:                        true,
	}
}

func processMessage(txn io.Txn, msg io.MsgDecoded) error {
	if msg.IsDeleted() {
		return processDeletingMessage(txn, msg)
	}
	return processCUMessage(txn, msg)
}

func processCUMessage(txn io.Txn, msg io.MsgDecoded) error {
	jwks := &model.JWKS{}

	err := json.Unmarshal(msg.Data, jwks)
	if err != nil {
		return err
	}

	return txn.Insert(model.JWKSType, jwks)
}

func processDeletingMessage(txn io.Txn, msg io.MsgDecoded) error {
	obj, err := txn.First(model.JWKSType, "id", msg.ID)
	if err != nil {
		return err
	}
	if obj == nil {
		return nil
	}
	return txn.Delete(model.JWKSType, obj)
}
