package usecase

import (
	"context"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
)

func runJWKSTest(t *testing.T, b logical.Backend, storage logical.Storage) []jose.JSONWebKey {
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "jwks",
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	jwt.RequireValidResponse(t, resp, err)

	keys := resp.Data["keys"].([]jose.JSONWebKey)
	require.NoError(t, err, "error on keys unmarshall")

	return keys
}

func TestJWKS(t *testing.T) {
	b, storage, memstore := jwt.GetBackend(t, time.Now)
	jwt.EnableJWT(t, b, storage)

	// #1 First run
	firstRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 1, len(firstRunKeys))

	tnx := memstore.Txn(true)
	defer tnx.Abort()
	stateRepo := model.NewStateRepo(tnx, model.NewJWKSRepo(tnx, "id"))
	s := NewKeyPairService(stateRepo, model.DefaultConfig(), time.Now)

	// #2 Second run (do not rotate first key

	err := s.GenerateOrRotateKeys()
	require.NoError(t, err)

	secondRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 2, len(secondRunKeys))
	require.Equal(t, secondRunKeys[0], firstRunKeys[0])

	// #3 Third run (delete first keys, add new key to the end)
	err = s.GenerateOrRotateKeys()
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
	jwt.RequireValidResponse(t, resp, err)
	keysAfterRotation := runJWKSTest(t, b, storage)
	require.Equal(t, len(keysAfterRotation), 1)
}

func TestJWKSRotation(t *testing.T) {
	b, storage, memstore := jwt.GetBackend(t, time.Now)
	jwt.EnableJWT(t, b, storage)

	// #1 First run
	firstRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, len(firstRunKeys), 1)

	tnx := memstore.Txn(true)
	defer tnx.Abort()
	stateRepo := model.NewStateRepo(tnx, model.NewJWKSRepo(tnx, "id"))
	s := NewKeyPairService(stateRepo, model.DefaultConfig(), time.Now)

	// #2 Nothing to do
	err := s.RunPeriodicalRotateKeys()
	require.NoError(t, err)

	secondRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, firstRunKeys, secondRunKeys)

	s = NewKeyPairService(stateRepo, model.DefaultConfig(), func() time.Time {
		return time.Now().Add(time.Hour * 330)
	})

	err = s.RunPeriodicalRotateKeys()
	require.NoError(t, err)

	thirdRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 2, len(thirdRunKeys))
	require.Equal(t, thirdRunKeys[0], firstRunKeys[0])


	s = NewKeyPairService(stateRepo, model.DefaultConfig(), func() time.Time {
		return time.Now().Add(time.Hour * 1000)
	})

	// #4 Rotation time
	err = s.RunPeriodicalRotateKeys()
	require.NoError(t, err)

	forthRunKeys := runJWKSTest(t, b, storage)
	require.Equal(t, 1, len(forthRunKeys))
	require.Equal(t, forthRunKeys[0], thirdRunKeys[1])
}
