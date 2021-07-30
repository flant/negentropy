package kafka_source

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type RestoreFunc func(*memdb.Txn, string, []byte) (bool, error)

type SelfKafkaSource struct {
	kf              *sharedkafka.MessageBroker
	decryptor       *sharedkafka.Encrypter
	restoreHandlers []RestoreFunc
}

func NewSelfKafkaSource(kf *sharedkafka.MessageBroker, restoreHandlers []RestoreFunc) *SelfKafkaSource {
	return &SelfKafkaSource{
		kf:        kf,
		decryptor: sharedkafka.NewEncrypter(),

		restoreHandlers: restoreHandlers,
	}
}

func (mks *SelfKafkaSource) Name() string {
	return mks.kf.PluginConfig.SelfTopicName
}

func (mks *SelfKafkaSource) Restore(txn *memdb.Txn) error {
	r := mks.kf.GetRestorationReader(mks.kf.PluginConfig.SelfTopicName)
	defer r.Close()

	return sharedkafka.RunRestorationLoop(r, nil, mks.kf.PluginConfig.SelfTopicName, txn, mks.restorationHandler)
}

func (mks *SelfKafkaSource) restorationHandler(txn *memdb.Txn, msg *kafka.Message) error {
	splitted := strings.Split(string(msg.Key), "/")
	if len(splitted) != 2 {
		return fmt.Errorf("key has wong format: %s", string(msg.Key))
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
		return err
	}

	hashed := sha256.Sum256(decrypted)
	err = rsa.VerifyPKCS1v15(mks.kf.EncryptionPublicKey(), crypto.SHA256, hashed[:], signature)
	if err != nil {
		return err
	}

	for _, r := range mks.restoreHandlers {
		handled, err := r(txn, splitted[0], decrypted)
		if err != nil {
			return err
		}

		if handled {
			return nil
		}
	}

	// Fill here objects for unmarshalling
	var inputObject interface{}
	switch splitted[0] {
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
		// return model.NewUserRepository(txn).Sync(splitted[1], decrypted)
		inputObject = &model.User{}

	case model.MultipassType:
		inputObject = &model.Multipass{}

	case model.ServiceAccountPasswordType:
		inputObject = &model.ServiceAccountPassword{}

	default:
		return errors.New("is not implemented yet")
	}

	err = json.Unmarshal(decrypted, inputObject)
	if err != nil {
		return err
	}

	err = txn.Insert(splitted[0], inputObject)
	if err != nil {
		return err
	}

	return nil
}

func (mks *SelfKafkaSource) Run(store *io.MemoryStore) {
	// do nothing
}

func (mks *SelfKafkaSource) Stop() {
	// do nothing
}
