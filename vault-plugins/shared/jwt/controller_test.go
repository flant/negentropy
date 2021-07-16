package jwt

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/test"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
	"strings"
	"testing"
	"time"
)

// Simple controller for test purposes (treat it like an example)
type jwtAuthBackend struct {
	*framework.Backend
	controller *Controller
}

func getBackend(t *testing.T, now func() time.Time) (*jwtAuthBackend, logical.Storage, *sharedio.MemoryStore) {
	b := new(jwtAuthBackend)

	conf := test.PrepareBackend(t)
	storage := test.GetStorage(t, conf)

	b.controller = NewJwtController(storage, func() (string, error) {
		return "id", nil
	}, log.NewNullLogger(), now)

	b.Backend = &framework.Backend{
		BackendType:  logical.TypeCredential,
		PathsSpecial: &logical.Paths{},
		Paths: framework.PathAppend(
			b.controller.ApiPaths(),
		),
	}

	err := b.Setup(context.Background(), conf)
	require.NoError(t, err)

	return b, conf.StorageView, storage
}

func TestIssue(t *testing.T){
	b, storage, memStore := getBackend(t, func() time.Time {
		return time.Unix(1619592212, 0)
	})

	test.EnableJWT(t, b, storage)

	{
		options := usecase.PrimaryTokenOptions{
			TTL:  10 * time.Minute,
			UUID: "test",
			JTI: usecase.TokenJTI {
				Generation: 0,
				SecretSalt: "test",
			},
		}

		tnx := memStore.Txn(false)
		defer tnx.Abort()
		token, err := b.controller.IssueMultipass(tnx, &options)
		require.NoError(t, err)

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "jwks",
			Storage:   storage,
			Data:      nil,
		}
		resp, err := b.HandleRequest(context.Background(), req)
		test.RequireValidResponse(t, resp, err)

		keys := resp.Data["keys"].([]jose.JSONWebKey)
		require.NoError(t, err, "error on keys unmarshall")

		pubKey := keys[0]
		jsonWebSig, err := jose.ParseSigned(token)
		require.NoError(t, err)

		_, err = jsonWebSig.Verify(pubKey.Key.(ed25519.PublicKey))
		require.NoError(t, err)

		payload := strings.Split(token, ".")[1]
		decodedPayload, err := base64.StdEncoding.DecodeString(payload + "=")
		require.NoError(t, err)

		var issuedToken usecase.PrimaryTokenClaims
		err = json.Unmarshal(decodedPayload, &issuedToken)
		require.NoError(t, err)

		require.Equal(t, usecase.PrimaryTokenClaims{
			TokenClaims: usecase.TokenClaims{
				IssuedAt: 1619592212,
				Expiry:   1619592812,
				Issuer:   "https://auth.negentropy.flant.com/",
			},
			Audience: "",
			Subject:  "test",
			JTI:      "8dea54dbe241bb7c6e9da12c6df39fbab2b76b6ad04c70f889d14f516df49a26", // "0 test"

		}, issuedToken)
	}
}