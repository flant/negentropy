package backend

import (
	"context"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/test"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"testing"
	"time"

	"github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
)

// Simple backend for test purposes (treat it like an example)
type jwtAuthBackend struct {
	*framework.Backend
	backend *Backend
}

func getBackend(t *testing.T, now func() time.Time) (*jwtAuthBackend, logical.Storage, *sharedio.MemoryStore) {
	b := new(jwtAuthBackend)

	conf := test.PrepareBackend(t)
	storage := test.GetStorage(t, conf)

	deps := usecase.NewDeps(func() (string, error) {
		return "id", nil
	}, log.NewNullLogger(), now)
	b.backend = NewBackend(storage, deps)

	b.Backend = &framework.Backend{
		BackendType:  logical.TypeCredential,
		PathsSpecial: &logical.Paths{},
		Paths: framework.PathAppend([]*framework.Path{
			PathRotateKey(b.backend),
			PathDisable(b.backend),
			PathEnable(b.backend),
			PathJWKS(b.backend),
			PathConfigure(b.backend),
		}),
	}

	err := b.Setup(context.Background(), conf)
	require.NoError(t, err)

	return b, conf.StorageView, storage
}

func TestJWTConfigure(t *testing.T) {
	now := func() time.Time { return time.Unix(1619592212, 0) }
	b, storage, _ := getBackend(t, now)
	test.EnableJWT(t, b, storage, true)

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
		test.RequireValidResponse(t, resp, err)

		require.Equal(t, map[string]interface{}{
			"issuer":                      "https://auth.negentropy.flant.com/",
			"multipass_audience":                "",
			"preliminary_announce_period": "24h0m0s",
			"rotation_period":             "336h0m0s",
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
				"multipass_audience":          "test",
				"preliminary_announce_period": "1h",
				"rotation_period":             "1h",
			},
		}

		resp, err := b.HandleRequest(context.Background(), req)
		test.RequireValidResponse(t, resp, err)
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
		test.RequireValidResponse(t, resp, err)

		require.Equal(t, map[string]interface{}{
			"issuer":                      "https://test",
			"multipass_audience":          "test",
			"preliminary_announce_period": "1h0m0s",
			"rotation_period":             "1h0m0s",
		}, resp.Data)
	}
}
