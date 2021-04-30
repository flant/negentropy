package kafka

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalingBrokerConfig(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		var config BrokerConfig

		data := []byte("{}")
		err := json.Unmarshal(data, &config)
		require.NoError(t, err)
	})

	t.Run("existent config", func(t *testing.T) {
		priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		rpriv, err := rsa.GenerateKey(rand.Reader, 4096)
		require.NoError(t, err)

		storedConfig := BrokerConfig{
			Endpoints:             []string{"localhost:9093", "localhost:9094"},
			ConnectionPrivateKey:  priv,
			ConnectionCertificate: nil,
			EncryptionPrivateKey:  rpriv,
			EncryptionPublicKey:   &rpriv.PublicKey,
		}

		d2, err := json.Marshal(storedConfig)
		require.NoError(t, err)

		var newConfig BrokerConfig

		err = json.Unmarshal(d2, &newConfig)
		require.NoError(t, err)
		assert.Equal(t, storedConfig.Endpoints, newConfig.Endpoints)
		assert.Equal(t, storedConfig.ConnectionCertificate, newConfig.ConnectionCertificate)
		assert.Equal(t, storedConfig.ConnectionPrivateKey, newConfig.ConnectionPrivateKey)
		assert.Equal(t, storedConfig.EncryptionPublicKey, newConfig.EncryptionPublicKey)
		assert.Equal(t, storedConfig.EncryptionPrivateKey, newConfig.EncryptionPrivateKey)
	})

}
