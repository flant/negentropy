package internal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

const serverKafka = "localhost:9094"

type mockPrceeder struct {
	t       *testing.T
	counter int
	ks      *NegentropyKafkaSource
}

func (m *mockPrceeder) ProceedMessage(key, decrypted []byte) error {
	splittedKey := strings.Split(string(key), "/")
	objID := splittedKey[1]
	if len(objID) > 32 {
		objID = objID[:32]
	}
	objectJson := string(decrypted)
	require.Truef(m.t, strings.Contains(objectJson, objID), fmt.Sprintf("object ID :'%s' should be in object:%s", objID, objectJson))
	m.counter += 1
	if m.counter > 100 {
		go func() {
			m.ks.Stop()
		}()
	}
	return nil
}

const rootPublicKey = "-----BEGIN RSA PUBLIC KEY-----\nMIICCgKCAgEAt433dLrnZpAapoJMLWa3DJY21evApUqpDhVvyjLd9izbaDNwLOhv\nSITuSw/M56GvxC5KlQoZdJ24zMI+gAqJ9Y2UJcbCkCIT7gpBBJcc/OSbIppmJXj/\nIgmSdeLaqneRwnR0YLSmOqJP4/o53B/ljmDLCBjhum98YfPyIUgQei5OUhcsoK4D\nfiIqTHUDuC4ch8dxuG7yKimmBT87JExU/zZvQPDWl37m94KvnVX6DFW7CwicWkja\n7IMTGSXnf7eiICqTo1N+JZLYLNPzvLw4mP1vURm2StZGva40mqgEWZsLxiMX1g0R\nXkDbT2KluUwv7KXtWPU/57crb8Kme/ak5iLTIGEIy10+I/7JKBSlPmQoflhS4+7z\nB0b2pca9H3WpVp2gczMbBbZBdGpCpgsvIOH+ABhh/KQSLaIXZqO2I6vuTaogkZGD\nOAWS+w9uDVuTuqIwM66vkTXxkm7vTaUUquoZe4SfzJsEz9QPNbKas1Px1P50uEg9\nu9QqmbmywDDCgL+8NPHsYMtCL5yAOkBQIMcppMPX+RH/GwrRqw7CsnwuGb7S2x/r\nH5Gy966Z53235UK/gGd82mz/4Jgfc7nVctl+3dGno4ACeFp7GF0UZ/HbgBOZ6qnQ\ngsVfG5SDlzxZ0dasF+NVvYf2/pUDJPK91av/3dI5ale54svpOSTHk1ECAwEAAQ==\n-----END RSA PUBLIC KEY-----\n"
const consumerPrivateKey = "-----BEGIN RSA PRIVATE KEY-----\n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\n-----END RSA PRIVATE KEY-----"

func Test_readingKafka(t *testing.T) {
	publicKey, err := ParseRSAPubKey(rootPublicKey)
	require.NoError(t, err)
	privateKey, err := ParseRSAPrivateKey(consumerPrivateKey)
	require.NoError(t, err)

	logger := hclog.NewNullLogger()
	mock := &mockPrceeder{t: t}
	kfs, err := NewKafkaSource(
		sharedkafka.BrokerConfig{
			Endpoints: []string{serverKafka},
			SSLConfig: &sharedkafka.SSLConfig{
				UseSSL:                true,
				CAPath:                "/Users/admin/flant/negentropy/docker/kafka/ca.crt",
				ClientPrivateKeyPath:  "/Users/admin/flant/negentropy/docker/kafka/client.key",
				ClientCertificatePath: "/Users/admin/flant/negentropy/docker/kafka/client.crt",
			},
			EncryptionPrivateKey: privateKey, // Here should be private part of replica key
			EncryptionPublicKey:  publicKey,  // Here should be public key from  flant_iam
		},
		"root_source.bush",
		"bush_consumer11",
		logger,
		mock,
	)
	mock.ks = kfs
	require.NoError(t, err)
	kfs.Run()
}
