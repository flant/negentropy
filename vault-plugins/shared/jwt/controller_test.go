package jwt

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/test"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
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

func verifyAndGetTokensTest(t *testing.T, keys []jose.JSONWebKey, token string) map[string]interface{} {
	parsed, err := jwt.ParseSigned(token)
	require.NoError(t, err)

	dest := map[string]interface{}{}
	err = parsed.Claims(keys[0], &dest)

	require.NoError(t, err)

	return dest
}

func assertRequiredTokenFields(t *testing.T, data map[string]interface{}, conf *model.Config, o *usecase.TokenOptions, nowF func() time.Time) {
	require.Contains(t, data, "iss")
	require.Equal(t, data["iss"], conf.Issuer)

	now := float64(nowF().Unix())
	ttl := o.TTL.Seconds()

	require.Contains(t, data, "iat")
	require.LessOrEqual(t, data["iat"], now)
	require.Less(t, data["iat"], now+2.0)

	require.Contains(t, data, "exp")
	require.Equal(t, data["exp"], data["iat"].(float64)+ttl)
}

func TestIssueMultipass(t *testing.T) {
	options := usecase.PrimaryTokenOptions{
		TTL:  10 * time.Minute,
		UUID: "test",
		JTI: usecase.TokenJTI{
			Generation: 0,
			SecretSalt: "test",
		},
	}

	now := func() time.Time {
		return time.Unix(1619592212, 0)
	}

	t.Run("does not issue multipass if jwt is disabled", func(t *testing.T) {
		b, _, memStore := getBackend(t, now)
		txn := memStore.Txn(false)
		defer txn.Abort()

		token, err := b.controller.IssueMultipass(txn, &options)

		require.Error(t, err)
		require.Empty(t, token)
	})

	t.Run("does not issue multipass after disable jwt", func(t *testing.T) {
		b, storage, memStore := getBackend(t, now)

		test.EnableJWT(t, b, storage, true)

		txn := memStore.Txn(false)
		defer txn.Abort()

		token, err := b.controller.IssueMultipass(txn, &options)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		test.EnableJWT(t, b, storage, false)

		token, err = b.controller.IssueMultipass(txn, &options)
		require.Error(t, err)
		require.Empty(t, token)
	})
	t.Run("issues correct multipass if jwt is enbled", func(t *testing.T) {
		b, storage, memStore := getBackend(t, now)
		test.EnableJWT(t, b, storage, true)

		txn := memStore.Txn(false)
		defer txn.Abort()
		token, err := b.controller.IssueMultipass(txn, &options)
		require.NoError(t, err)

		k, err := b.controller.JWKS(txn)
		require.Len(t, k, 1)

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
	})
}

func Test_NewToken(t *testing.T) {
	now := time.Now

	b, storage, memstore := getBackend(t, now)
	test.EnableJWT(t, b, storage, true)
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "jwks",
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	test.RequireValidResponse(t, resp, err)

	keys := resp.Data["keys"].([]jose.JSONWebKey)

	t.Run("signing payload", func(t *testing.T) {
		const ttl = 5

		tokenOpt := &usecase.TokenOptions{
			TTL: time.Duration(ttl) * time.Second,
		}

		txn := memstore.Txn(false)
		defer txn.Abort()
		conf, err := b.controller.GetConfig(txn)
		require.NoError(t, err)

		t.Run("signs payload successfully", func(t *testing.T) {
			payload := map[string]interface{}{
				"aud": "Aud",
				"payload_map": map[string]interface{}{
					"a": "a_val",
					"b": float64(1),
				},
			}
			token, err := b.controller.IssuePayloadAsJwt(txn, payload, tokenOpt)
			require.NoError(t, err)

			data := verifyAndGetTokensTest(t, keys, token)

			for k, v := range payload {
				require.Contains(t, data, k)
				require.Equal(t, data[k], v)
			}

			assertRequiredTokenFields(t, data, conf, tokenOpt, now)
		})

		t.Run("signs does not override issuer expiration time and issue time", func(t *testing.T) {
			payload := map[string]interface{}{
				"aud": "Aud",
				"payload_map": map[string]interface{}{
					"a": "a_val",
					"b": float64(1),
				},
				"iss": "pirate issuer",
				"iat": 20,
				"exp": 100500,
			}
			token, err := b.controller.IssuePayloadAsJwt(txn, payload, tokenOpt)
			require.NoError(t, err)

			data := verifyAndGetTokensTest(t, keys, token)

			assertRequiredTokenFields(t, data, conf, tokenOpt, now)
		})
	})
}
