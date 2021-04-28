package kafka_source

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"log"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-memdb"

	self "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/handlers/self"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type SelfKafkaSource struct {
	kf        *kafka.MessageBroker
	decryptor *kafka.Encrypter
	topic     string

	api *vault.VaultEntityDownstreamApi
}

func NewSelfKafkaSource(kf *kafka.MessageBroker, api *vault.VaultEntityDownstreamApi) *SelfKafkaSource {
	return &SelfKafkaSource{
		kf:        kf,
		decryptor: kafka.NewEncrypter(),
		topic:     kf.PluginConfig.SelfTopicName,
		api:        api,
	}
}

func (sks *SelfKafkaSource) Restore(txn *memdb.Txn) error {
	r := sks.kf.GetRestorationReader(sks.kf.PluginConfig.SelfTopicName)
	defer r.Close()

	replicaName := sks.kf.PluginConfig.SelfTopicName

	runConsumer := sks.kf.GetConsumer(replicaName, replicaName, false)
	_, lastOffset, err := runConsumer.GetWatermarkOffsets(replicaName, 0)
	if err != nil {
		return err
	}
	_ = runConsumer.Close()

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

		decrypted, err := sks.decryptData(m.Value, chunked)
		if err != nil {
			log.Printf("can't decrypt message. Skipping: %s in topic: %s at offset %d\n", m.Key, *m.TopicPartition.Topic, m.TopicPartition.Offset)
			continue
		}

		if len(signature) == 0 {
			log.Printf("no signature found. Skipping message: %s in topic: %s at offset %d\n", m.Key, *m.TopicPartition.Topic, m.TopicPartition.Offset)
			continue
		}

		err = sks.verifySign(signature, decrypted)
		if err != nil {
			log.Printf("wrong signature. Skipping message: %s in topic: %s at offset %d\n", m.Key, *m.TopicPartition.Topic, m.TopicPartition.Offset)
			continue
		}

		err = self.HandleRestoreMessagesSelfSource(txn, splitted[0], decrypted)
		if err != nil {
			return err
		}

		if int64(m.TopicPartition.Offset) == lastOffset-1 {
			return nil
		}
	}
}

func (sks *SelfKafkaSource) Run(store *io.MemoryStore) {
	replicaName := sks.kf.PluginConfig.SelfTopicName
	rd := sks.kf.GetConsumer(replicaName, replicaName, false)

	for {
		msg, err := rd.ReadMessage(-1)
		if err != nil {
			// TODO: shouldn't we fail?
			log.Println("read message failed", err)
			continue
		}

		splitted := strings.Split(string(msg.Key), "/")
		if len(splitted) != 2 {
			// TODO: ??
			// return fmt.Errorf("key has wong format: %s", string(msg.Key))
			continue
		}
		objType := splitted[0]

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

		decrypted, err := sks.decryptData(msg.Value, chunked)
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

		err = sks.verifySign(signature, decrypted)
		if err != nil {
			log.Printf("wrong signature. Skipping message: %s in topic: %s at offset %d\n",
				msg.Key, *msg.TopicPartition.Topic, msg.TopicPartition.Offset)
			continue
		}

		source, err := kafka.NewSourceInputMessage(rd, msg.TopicPartition)
		if err != nil {
			log.Println("build source message failed", err)
		}

		operation := func() error {
			return sks.processMessage(source, store, objType, decrypted)
		}
		err = backoff.Retry(operation, backoff.NewExponentialBackOff())
		if err != nil {
			panic(err)
		}
	}
}

func (sks *SelfKafkaSource) processMessage(source *kafka.SourceInputMessage, store *io.MemoryStore, objType string, data []byte) error {
	tx := store.Txn(true)
	defer tx.Abort()

	err := self.HandleNewMessageSelfSource(tx, self.NewObjectHandler(store, tx, sks.api), objType, data)
	if err != nil {
		return err
	}
	return tx.Commit(source)
}

func (sks *SelfKafkaSource) decryptData(data []byte, chunked bool) ([]byte, error) {
	return sks.decryptor.Decrypt(data, sks.kf.EncryptionPrivateKey(), chunked)
}

func (sks *SelfKafkaSource) verifySign(signature []byte, data []byte) error {
	hashed := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(sks.kf.EncryptionPublicKey(), crypto.SHA256, hashed[:], signature)
}
