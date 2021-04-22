package kafka_source

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/go-memdb"

	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type MainKafkaSource struct {
	kf *sharedkafka.MessageBroker

	topic string
}

func NewMainKafkaSource(kf *sharedkafka.MessageBroker, topic string) MainKafkaSource {
	return MainKafkaSource{kf: kf, topic: topic}
}

func (mks MainKafkaSource) Restore(txn *memdb.Txn) error {
	r := mks.kf.GetRestorationReader("", "root_source")
	defer r.Close()

	// we dont have other consumers on this topic. Get MaxOffset from its single partition
	lastOffset, err := mks.kf.GetLastOffset("root_source")
	if err != nil {
		return err
	}

	if lastOffset <= 0 {
		return nil
	}

	for {
		m, err := r.ReadMessage(context.TODO())
		if err != nil {
			return err
		}

		splitted := strings.Split(string(m.Key), "/")
		if len(splitted) != 2 {
			return fmt.Errorf("key has wong format: %s", string(m.Key))
		}

		decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, mks.kf.EncryptionPrivateKey(), m.Value)
		if err != nil {
			return err
		}

		var signature []byte
		for _, header := range m.Headers {
			if header.Key == "signature" {
				signature = header.Value
			}
		}

		hashed := sha256.Sum256(decrypted)
		err = rsa.VerifyPKCS1v15(mks.kf.EncryptionPublicKey(), crypto.SHA256, hashed[:], signature)
		if err != nil {
			return err
		}

		// blyat
		// TODO: need huge switch-case here, with object Unmarshaling
		// var foo FooBar
		// err = json.Unmarshal(m.Value, &foo)
		// if err != nil {
		// 	return err
		// }
		var foo interface{}
		err = json.Unmarshal(decrypted, &foo)
		if err != nil {
			return err
		}

		err = txn.Insert(splitted[0], foo)
		if err != nil {
			return err
		}

		if m.Offset == lastOffset-1 {
			return nil
		}
	}
}
