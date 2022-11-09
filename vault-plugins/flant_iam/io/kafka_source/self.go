package kafka_source

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	log "github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type RestoreFunc func(txn io.Txn, decoded sharedkafka.MsgDecoded) (handled bool, err error)

func NewSelfKafkaSource(kf *sharedkafka.MessageBroker, restoreHandlers []RestoreFunc, parentLogger log.Logger) *io.KafkaSourceImpl {
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
	proccessRestoreMessage := func(txn io.Txn, m sharedkafka.MsgDecoded) error {
		for _, r := range restoreHandlers {
			handled, err := r(txn, m)
			if err != nil {
				return err
			}

			if handled {
				return nil
			}
		}

		// Fill here objects for unmarshalling
		var inputObject interface{}
		objectType := m.Type
		switch objectType {
		case model.ReplicaType:
			inputObject = &model.Replica{}

		case model.FeatureFlagType:
			inputObject = &model.FeatureFlag{}

		case model.GroupType:
			inputObject = &model.Group{}

		case model.ProjectType:
			inputObject = &model.Project{}

		case model.RoleType:
			inputObject = &model.Role{}

		case model.RoleBindingType:
			inputObject = &model.RoleBinding{}

		case model.ServiceAccountType:
			inputObject = &model.ServiceAccount{}

		case model.TenantType:
			inputObject = &model.Tenant{}

		case model.UserType:
			inputObject = &model.User{}

		case model.MultipassType:
			inputObject = &model.Multipass{}

		case model.ServiceAccountPasswordType:
			inputObject = &model.ServiceAccountPassword{}

		case model.IdentitySharingType:
			inputObject = &model.IdentitySharing{}

		case model.RoleBindingApprovalType:
			inputObject = &model.RoleBindingApproval{}

		default:
			return fmt.Errorf("type= %s: is not implemented yet", objectType)
		}

		err := json.Unmarshal(m.Data, inputObject)
		if err != nil {
			return err
		}

		return txn.Insert(objectType, inputObject)
	}

	return &io.KafkaSourceImpl{
		NameOfSource:              "iamSelfKafkaSource",
		KafkaBroker:               kf,
		Logger:                    parentLogger.Named("iamSelfKafkaSource"),
		ProvideRunConsumerGroupID: runConsumerGroupIDProvider,
		ProvideTopicName:          topicNameProvider,
		VerifySign:                verifySign,
		Decrypt:                   decrypt,
		ProcessRunMessage:         nil, // don't need as not runnable
		ProcessRestoreMessage:     proccessRestoreMessage,
		Runnable:                  false, // no need run
	}
}
