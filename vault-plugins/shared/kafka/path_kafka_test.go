package kafka

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"

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
		tb.broker.config.EncryptionPrivateKey = r
		tb.broker.config.EncryptionPublicKey = &r.PublicKey

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
			dd, _ := json.Marshal(tb.broker.config)
			err = storage.Put(cctx, &logical.StorageEntry{Key: kafkaConfigPath, Value: dd, SealWrap: true})
			require.NoError(t, err)
			mb, err := NewMessageBroker(cctx, storage, "test")
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

func TestConfigureAccess(t *testing.T) {
	t.Run("invalid certificate", func(t *testing.T) {
		b, storage := generateBackend(t)
		tb := b.(testBackend)
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		tb.broker.config.ConnectionPrivateKey = priv

		cert := `-----BEGIN CERTIFICATE-----
MIICdjCCAV6gAwIBAgIBDTANBgkqhkiG9w0BAQUFADAVMRMwEQYDVQQDDAo4Z3dp
Zmkub3JnMB4XDTIxMDQyMjE5NTczNloXDTIxMDUwNjEzNDQxMVowFDESMBAGA1UE
AwwJZmxhbnRfaWFtMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAETXid30za8SJc
06pLzrNDrc7q8FKu/JbHUUoS4Uu6vsGjiA2gtalIX/L2/frp3m87OC3R7MSDJMfc
bLUB1drOxaOBnDCBmTA8BgNVHSMENTAzgBQh/Rv9y18DPkG4uoVpWoySLuNJtKEV
pBMwETEPMA0GA1UEAwwGcm9vdENBggSc0xydMB0GA1UdDgQWBBTP5pfsl1LXX/6o
W+jrxLmKiVAQejASBgNVHRMBAf8ECDAGAQH/AgEAMA4GA1UdDwEB/wQEAwIFoDAW
BgNVHSUBAf8EDDAKBggrBgEFBQcDATANBgkqhkiG9w0BAQUFAAOCAQEAL/pS68m4
1RLAbEYvGfBt0ulgoJqYq7yuE04w0F4oB5fCRoworY1MT/WtyBIdcyXa9DEd3EAN
kYxKR+OOvCLvVILomleugxDV8dU83mWN9FMZ7iFJM012Es+3TMPuAnuV2q0YyvU0
S0iRtml6tAFNn17klePEs0NVSokW+xrCfOmYKmIxJ0/+la8HHboFTihJjKFBfuXV
FMVYexsfRuqZKx/hJqGr7EgRVY2zLY+/4rIvYc7oZfk6t4OF4BHGQQzoeQY/rbkz
/G++/aEn7hsTUOoUqZGO9knaR2G94Ca2zvUIkt7aGsfV880mTXAd8hpU6lCFfbAy
VuIXvfOaQNX7wA==
-----END CERTIFICATE-----`

		req := &logical.Request{
			Storage:   storage,
			Operation: logical.UpdateOperation,
			Path:      "kafka/configure_access",
			Data:      map[string]interface{}{"certificate": cert, "kafka_endpoints": []string{"192.168.1.1:3434"}},
		}

		_, err = b.HandleRequest(cctx, req)
		require.Error(t, err)
		assert.Equal(t, "tls: private key does not match public key", err.Error())
	})
}

func TestGenerateCSR(t *testing.T) {
	b, storage := generateBackend(t)

	req := &logical.Request{
		Storage:   storage,
		Operation: logical.CreateOperation,
		Path:      "kafka/generate_csr",
	}

	resp, err := b.HandleRequest(cctx, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)

	t.Run("second run warning", func(t *testing.T) {
		resp, err := b.HandleRequest(cctx, req)
		fmt.Println(resp.Warnings)
		require.NoError(t, err)
		assert.Equal(t, []string{"Private key is already exist. Add ?force=true param to recreate it"}, resp.Warnings)
	})
}
