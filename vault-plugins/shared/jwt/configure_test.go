package jwt

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"
	"gopkg.in/square/go-jose.v2"
)

func TestJWTConfigure(t *testing.T) {
	b, storage := getBackend(t)
	enableJWT(t, b, storage)

	const jwtConfigurePath = "jwt/configure"

	// #1 Read the config
	{
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      jwtConfigurePath,
			Storage:   storage,
			Data:      nil,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		requireValidResponse(t, resp, err)

		require.Equal(t, map[string]interface{}{
			"issuer":                      "https://auth.negentropy.flant.com/",
			"own_audience":                "",
			"preliminary_announce_period": "24h",
			"rotation_period":             "336h",
		}, resp.Data)
	}

	// #2 Configure non default values
	{
		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      jwtConfigurePath,
			Storage:   storage,
			Data: map[string]interface{}{
				"issuer":                      "https://test",
				"own_audience":                "test",
				"preliminary_announce_period": "1h",
				"rotation_period":             "1h",
			},
		}

		resp, err := b.HandleRequest(context.Background(), req)
		requireValidResponse(t, resp, err)
	}

	// #3 Check again
	{
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      jwtConfigurePath,
			Storage:   storage,
			Data:      nil,
		}

		resp, err := b.HandleRequest(context.Background(), req)
		requireValidResponse(t, resp, err)

		require.Equal(t, map[string]interface{}{
			"issuer":                      "https://test",
			"own_audience":                "test",
			"preliminary_announce_period": "1h",
			"rotation_period":             "1h",
		}, resp.Data)
	}

	// #4 Generate and verify token
	{
		token, err := newPrimaryToken(context.TODO(), storage)
		require.NoError(t, err)

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

		pubKey := keys[0]
		jsonWebSig, err := jose.ParseSigned(token)
		require.NoError(t, err)

		_, err = jsonWebSig.Verify(pubKey.Key.(ed25519.PublicKey))
		require.NoError(t, err)
	}
}
