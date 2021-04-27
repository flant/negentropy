package jwt

import (
	"context"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
)

func getBackend(t *testing.T) (*jwtAuthBackend, logical.Storage) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(log.Trace),

		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}
	b, err := Factory(context.Background(), config)
	require.NoError(t, err)

	jwtBackend := b.(*jwtAuthBackend)
	return jwtBackend, config.StorageView
}

func enableJWT(t *testing.T, b logical.Backend, storage logical.Storage) {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "jwt/enable",
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	requireValidResponse(t, resp, err)
}

func requireValidResponse(t *testing.T, resp *logical.Response, err error) {
	require.NoError(t, err, "error on request")
	require.NotNil(t, resp, "empty response")
	require.False(t, resp.IsError(), "error response")
}

func runJWKSTest(t *testing.T, b logical.Backend, storage logical.Storage) []jose.JSONWebKey {
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "jwks",
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	requireValidResponse(t, resp, err)

	keys := resp.Data["keys"].([]jose.JSONWebKey)
	require.NoError(t, err, "error on keys unmarshall")

	return keys
}

func TestJWKS(t *testing.T) {
	b, storage := getBackend(t)
	enableJWT(t, b, storage)

	// #1 First run
	firstRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 1, len(firstRunKeys))

	// #2 Second run (do not rotate first key
	err := generateOrRotateKeys(context.TODO(), storage)
	require.NoError(t, err)
	secondRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 2, len(secondRunKeys))
	require.Equal(t, secondRunKeys[0], firstRunKeys[0])

	// #3 Third run (delete first keys, add new key to the end)
	err = generateOrRotateKeys(context.TODO(), storage)
	require.NoError(t, err)
	thisRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 2, len(thisRunKeys))
	require.Equal(t, thisRunKeys[0], secondRunKeys[1])

	// #4 Force rotate keys
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "jwt/rotate_key",
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	requireValidResponse(t, resp, err)
	keysAfterRotation := runJWKSTest(t, b, storage)
	require.Equal(t, len(keysAfterRotation), 1)
}

func TestJWKSRotation(t *testing.T) {
	b, storage := getBackend(t)
	enableJWT(t, b, storage)

	// #1 First run
	firstRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, len(firstRunKeys), 1)

	// #2 Nothing to do
	err := b.tokenController.rotateKeys(context.TODO(), &logical.Request{Storage: storage})
	require.NoError(t, err)

	secondRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, firstRunKeys, secondRunKeys)

	// #3 Publish time
	b.tokenController.now = func() time.Time {
		return time.Now().Add(time.Hour * 330)
	}
	err = b.tokenController.rotateKeys(context.TODO(), &logical.Request{Storage: storage})
	require.NoError(t, err)

	thirdRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 2, len(thirdRunKeys))
	require.Equal(t, thirdRunKeys[0], firstRunKeys[0])

	// #4 Rotation time
	b.tokenController.now = func() time.Time {
		return time.Now().Add(time.Hour * 1000)
	}
	err = b.tokenController.rotateKeys(context.TODO(), &logical.Request{Storage: storage})
	require.NoError(t, err)

	forthRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 1, len(forthRunKeys))
	require.Equal(t, forthRunKeys[0], thirdRunKeys[1])
}
