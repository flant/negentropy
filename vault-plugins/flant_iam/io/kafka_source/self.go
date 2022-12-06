package kafka_source

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	log "github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type RestoreFunc func(txn io.Txn, decoded io.MsgDecoded) (handled bool, err error)

func NewSelfKafkaSource(kf *sharedkafka.MessageBroker, restoreHandlers []RestoreFunc, parentLogger log.Logger) *io.KafkaSourceImpl {
	restoreHandlers = append(restoreHandlers, IamObjectsRestoreHandler)
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
	proccessRestoreMessage := func(txn io.Txn, m io.MsgDecoded) error {
		for _, r := range restoreHandlers {
			handled, err := r(txn, m)
			if err != nil {
				return err
			}

			if handled {
				return nil
			}
		}
		return fmt.Errorf("type= %s: %w", m.Type, consts.ErrNotHandledObject)
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

func IamObjectsRestoreHandler(txn io.Txn, m io.MsgDecoded) (bool, error) {
	handled, err := io.HandleTombStone(txn, m)
	if handled || err != nil {
		return handled, err
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
	}
	err = json.Unmarshal(m.Data, inputObject)
	if err != nil {
		return false, err
	}

	err = txn.Insert(objectType, inputObject)
	if err != nil {
		return false, err
	}
	return true, nil
}
