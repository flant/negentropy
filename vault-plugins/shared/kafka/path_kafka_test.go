package kafka

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cctx = context.Background()

func TestPublicKeyGet(t *testing.T) {
	t.Run("Kafka is not configured", func(t *testing.T) {
		t.Parallel()
		b, storage := generateBackend(t)

		req := &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      "kafka/public_key",
		}

		_, err := b.HandleRequest(cctx, req)
		require.Error(t, err)
		assert.Equal(t, err.Error(), "public key does not exist. Run /kafka/configure_access first")
	})

	t.Run("kafka configured", func(t *testing.T) {
		t.Parallel()
		b, storage := generateBackend(t)
		tb := b.(testBackend)

		r, err := rsa.GenerateKey(rand.Reader, 256)
		require.NoError(t, err)
		tb.broker.KafkaConfig.EncryptionPrivateKey = r
		tb.broker.KafkaConfig.EncryptionPublicKey = &r.PublicKey

		req := &logical.Request{
			Storage:   storage,
			Operation: logical.ReadOperation,
			Path:      "kafka/public_key",
		}

		resp, err := b.HandleRequest(cctx, req)
		require.NoError(t, err)
		keyPem := resp.Data["public_key"].(string)
		keyPem = strings.ReplaceAll(keyPem, "\\n", "\n")
		p, _ := pem.Decode([]byte(keyPem))
		kafkaPublicKey, err := x509.ParsePKCS1PublicKey(p.Bytes)
		require.NoError(t, err)
		assert.True(t, r.PublicKey.Equal(kafkaPublicKey))

		t.Run("after vault reboot", func(t *testing.T) {
			dd, _ := json.Marshal(tb.broker.KafkaConfig)
			err = storage.Put(cctx, &logical.StorageEntry{Key: kafkaConfigPath, Value: dd, SealWrap: true})
			require.NoError(t, err)
			mb, err := NewMessageBroker(cctx, storage, log.NewNullLogger())
			require.NoError(t, err)

			tb.broker = mb

			req := &logical.Request{
				Storage:   storage,
				Operation: logical.ReadOperation,
				Path:      "kafka/public_key",
			}

			resp, err := b.HandleRequest(cctx, req)
			require.NoError(t, err)
			keyPem := resp.Data["public_key"].(string)
			keyPem = strings.ReplaceAll(keyPem, "\\n", "\n")
			p, _ := pem.Decode([]byte(keyPem))
			kafkaPublicKey, err := x509.ParsePKCS1PublicKey(p.Bytes)
			require.NoError(t, err)
			assert.True(t, r.PublicKey.Equal(kafkaPublicKey))
		})
	})
}
