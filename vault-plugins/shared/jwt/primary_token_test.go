package jwt

import (
	"context"
	"gopkg.in/square/go-jose.v2/jwt"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
)

func verifyAndGetTokensTest(t *testing.T, keys []jose.JSONWebKey, token string) map[string]interface{} {
	parsed, err := jwt.ParseSigned(token)
	require.NoError(t, err)

	dest := map[string]interface{}{}
	err = parsed.Claims(keys[0], &dest)

	require.NoError(t, err)

	return dest
}

func assertRequiredTokenFields(t *testing.T, data map[string]interface{}, conf map[string]interface{}, o *TokenOptions) {
	require.Contains(t, data, "iss")
	require.Equal(t, data["iss"], conf["issuer"].(string))

	now := o.now().Unix()
	ttl := int64(o.TTL.Seconds())

	require.Contains(t, data, "iat")
	require.Equal(t, data["iat"], float64(now))

	require.Contains(t, data, "exp")
	require.Equal(t, data["exp"], float64(now+ttl))
}

func Test_NewToken(t *testing.T) {
	b, storage := getBackend(t)
	enableJWT(t, b, storage)
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "jwks",
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	requireValidResponse(t, resp, err)

	keys := resp.Data["keys"].([]jose.JSONWebKey)

	t.Run("signing payload", func(t *testing.T) {
		const ttl = 5
		const now = 1

		tokenOpt := &TokenOptions{
			TTL: time.Duration(ttl) * time.Second,
			now: func() time.Time {
				return time.Unix(now, 0)
			},
		}

		conf, err := getConfig(context.TODO(), storage)
		require.NoError(t, err)

		t.Run("signs payload successfully", func(t *testing.T) {
			payload := map[string]interface{}{
				"aud": "Aud",
				"payload_map": map[string]interface{}{
					"a": "a_val",
					"b": float64(1),
				},
			}
			token, err := NewJwtToken(context.TODO(), storage, payload, tokenOpt)
			require.NoError(t, err)

			data := verifyAndGetTokensTest(t, keys, token)

			for k, v := range payload {
				require.Contains(t, data, k)
				require.Equal(t, data[k], v)
			}

			assertRequiredTokenFields(t, data, conf, tokenOpt)
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
			token, err := NewJwtToken(context.TODO(), storage, payload, tokenOpt)
			require.NoError(t, err)

			data := verifyAndGetTokensTest(t, keys, token)

			assertRequiredTokenFields(t, data, conf, tokenOpt)

		})
	})
}
