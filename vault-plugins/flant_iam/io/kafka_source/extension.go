package kafka_source

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type ExtensionKafkaSource struct {
	kf        *sharedkafka.MessageBroker
	decryptor *sharedkafka.Encrypter

	extensionName string
	signKey       *rsa.PublicKey

	ownedTypes    map[string]struct{}
	extendedTypes map[string]struct{}
	allowedRoles  map[string]struct{}
}

func NewExtensionKafkaSource(kf *sharedkafka.MessageBroker, name string, pubKey *rsa.PublicKey, ownTypes, extTypes, roles []string) *ExtensionKafkaSource {
	ownedTypes := make(map[string]struct{}, len(ownTypes))
	extendedTypes := make(map[string]struct{}, len(extTypes))
	allowedRoles := make(map[string]struct{}, len(roles))

	for _, t := range ownTypes {
		ownedTypes[t] = struct{}{}
	}

	for _, t := range extTypes {
		extendedTypes[t] = struct{}{}
	}

	for _, t := range roles {
		allowedRoles[t] = struct{}{}
	}
	return &ExtensionKafkaSource{
		kf:        kf,
		decryptor: sharedkafka.NewEncrypter(),

		extensionName: name,
		signKey:       pubKey,

		ownedTypes:    ownedTypes,
		extendedTypes: extendedTypes,
		allowedRoles:  allowedRoles,
	}
}

func (mks *ExtensionKafkaSource) isAllowedType(typ string) bool {
	if _, ok := mks.ownedTypes[typ]; ok {
		return true
	}

	if _, ok := mks.extendedTypes[typ]; ok {
		return true
	}

	return false
}

func (mks *ExtensionKafkaSource) Restore(txn *memdb.Txn) error {
	r := mks.kf.GetRestorationReader(mks.kf.PluginConfig.SelfTopicName)
	defer r.Close()

	// we dont have other consumers on this topic. Get MaxOffset from its single partition
	_, lastOffset, err := r.GetWatermarkOffsets(mks.kf.PluginConfig.SelfTopicName, 0)
	if err != nil {
		return err
	}

	if lastOffset <= 0 {
		return nil
	}

	for {
		m, err := r.ReadMessage(-1)
		if err != nil {
			return err
		}

		splitted := strings.Split(string(m.Key), "/")
		if len(splitted) != 2 {
			return fmt.Errorf("key has wong format: %s", string(m.Key))
		}

		if !mks.isAllowedType(splitted[0]) {
			continue
		}

		var signature []byte
		var chunked bool
		for _, header := range m.Headers {
			switch header.Key {
			case "signature":
				signature = header.Value

			case "chunked":
				chunked = true
			}
		}

		decrypted, err := mks.decryptor.Decrypt(m.Value, mks.kf.EncryptionPrivateKey(), chunked)
		if err != nil {
			return err
		}

		hashed := sha256.Sum256(decrypted)
		err = rsa.VerifyPKCS1v15(mks.signKey, crypto.SHA256, hashed[:], signature)
		if err != nil {
			return err
		}

		var inputObject interface{}
		switch splitted[0] {
		case model.GroupType:
			inputObject = &model.Group{}

		case model.RoleBindingType:
			inputObject = &model.RoleBinding{}

		case model.ServiceAccountType:
			inputObject = &model.ServiceAccount{}

		case model.UserType:
			inputObject = &model.User{}

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

		if int64(m.TopicPartition.Offset) == lastOffset-1 {
			return nil
		}
	}
}

func (mks *ExtensionKafkaSource) Run(store *io.MemoryStore) {
	topicName := "extension." + mks.extensionName
	rd := mks.kf.GetConsumer(mks.extensionName, topicName, false)

	for {
		msg, err := rd.ReadMessage(-1)
		if err != nil {
			log.Printf("read message error: %s\n", err)
			continue
		}

		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			continue
		}
		objType, objID := splitted[0], splitted[1]

		if !mks.isAllowedType(objType) {
			continue
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

		decrypted, err := mks.decryptData(msg.Value, chunked)
		if err != nil {
			log.Printf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			continue
		}

		if len(signature) == 0 {
			log.Printf("no signature found. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			continue
		}

		err = mks.verifySign(signature, decrypted)
		if err != nil {
			log.Printf("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			continue
		}

		source, err := sharedkafka.NewSourceInputMessage(rd, msg.TopicPartition)
		if err != nil {
			log.Println("build source message failed", err)
			continue
		}

		operation := func() error {
			return mks.processMessage(source, store, objType, objID, decrypted)
		}
		err = backoff.Retry(operation, backoff.NewExponentialBackOff())
		if err != nil {
			panic(err)
		}
	}
}

func (mks *ExtensionKafkaSource) processMessage(source *sharedkafka.SourceInputMessage, store *io.MemoryStore, objType, objID string, data []byte) error {
	tx := store.Txn(true)

	switch objType {
	case model.UserType:
		// all logic here
		pl := &model.User{}
		_ = json.Unmarshal(data, pl)

		if _, ok := mks.ownedTypes[objType]; ok {
			// Process own type
			err := tx.Insert(model.UserType, pl)
			if err != nil {
				return backoff.Permanent(err)
			}
		}
	case model.ExtensionType:
		// extension:{uuid}
		if _, ok := mks.extendedTypes[objType]; !ok {
			return tx.Commit(source)
		}

		ext := &model.Extension{}
		_ = json.Unmarshal(data, ext)
		err := applyExtension(tx, ext)
		if err != nil {
			tx.Abort()
			return err
		}

	default:
		tx.Abort()
		return errors.New("not implemented yet")
	}
	return tx.Commit(source)
}

func (mks *ExtensionKafkaSource) decryptData(data []byte, chunked bool) ([]byte, error) {
	return mks.decryptor.Decrypt(data, mks.kf.EncryptionPrivateKey(), chunked)
}

func (mks *ExtensionKafkaSource) verifySign(signature []byte, data []byte) error {
	hashed := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(mks.signKey, crypto.SHA256, hashed[:], signature)
}

func applyExtension(db *io.MemoryStoreTxn, ext *model.Extension) error {
	switch ext.OwnerType {
	case model.UserType:
		repo := model.NewUserRepository(db)
		user, err := repo.GetById(ext.OwnerUUID)
		if err != nil {
			return fmt.Errorf("user not found: %v", err)
		}
		user.Extension = ext
		err = repo.Update(user)
		if err != nil {
			return fmt.Errorf("user not updated: %v", err)
		}
	case model.ServiceAccountType:
	case model.RoleBindingType:
	case model.GroupType:
		// TODO: case model.MultipassType:
	}
	return nil
}
