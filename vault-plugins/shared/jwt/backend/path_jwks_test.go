package backend

import (
	"context"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/shared/jwt/test"
)

func Test_Rotate(t *testing.T) {
	now := func() time.Time { return time.Unix(1619592212, 0) }
	b, storage, _ := getBackend(t, now)
	test.EnableJWT(t, b, storage, true)

	t.Run("successful rotate", func(t *testing.T) {
		reqJwks := func() map[string]interface{} {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      "jwks",
				Storage:   storage,
				Data:      nil,
			}

			resp, err := b.HandleRequest(context.Background(), req)
			test.RequireValidResponse(t, resp, err)
			return resp.Data
		}

		origJwks := reqJwks()

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "jwt/rotate_key",
			Storage:   storage,
			Data:      nil,
		}
		resp, err := b.HandleRequest(context.Background(), req)
		test.RequireValidResponse(t, resp, err)

		curJwks := reqJwks()

		diff := deep.Equal(origJwks, curJwks)
		require.NotNil(t, diff)
	})
}
