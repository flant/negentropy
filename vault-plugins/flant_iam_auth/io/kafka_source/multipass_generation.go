package kafka_source

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func NewMultipassGenerationSource(storage logical.Storage, kf *sharedkafka.MessageBroker, parentLogger hclog.Logger) *sharedio.KafkaSourceImpl {
	runConsumerGroupIDProvider := func(kf *sharedkafka.MessageBroker) string {
		return kf.PluginConfig.SelfTopicName
	}
	topicNameProvider := func(_ *sharedkafka.MessageBroker) string {
		return io.MultipassNumberGenerationTopic
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

	return &sharedio.KafkaSourceImpl{
		NameOfSource:                   "authMultipassKafkaSource",
		KafkaBroker:                    kf,
		Logger:                         parentLogger.Named("authMultipassKafkaSource"),
		ProvideRunConsumerGroupID:      runConsumerGroupIDProvider,
		ProvideTopicName:               topicNameProvider,
		VerifySign:                     verifySign,
		Decrypt:                        nil, // this common topic is not encrypted
		ProcessRunMessage:              processMessage,
		ProcessRestoreMessage:          processMessage,
		IgnoreSourceInputMessageBody:   true, // this topic has unusual Commit mechanic
		Runnable:                       true,
		RestoreStrictlyTillRunConsumer: true, // restore strictly to offset read by run consumer
		Storage:                        storage,
	}
}

func processMessage(txn sharedio.Txn, msg sharedio.MsgDecoded) error {
	handled, err := sharedio.HandleTombStone(txn, msg)
	if handled || err != nil {
		return err
	}
	return processCUMessage(txn, msg)
}

func processCUMessage(txn sharedio.Txn, msg sharedio.MsgDecoded) error {
	mpgn := &model.MultipassGenerationNumber{}

	err := json.Unmarshal(msg.Data, mpgn)
	if err != nil {
		return err
	}

	return txn.Insert(model.MultipassGenerationNumberType, mpgn)
}
