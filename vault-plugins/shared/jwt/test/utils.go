package test

import (
	"context"
	"testing"
	"time"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"

	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/backend"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

// Simple backend for test purposes (treat it like an example)
type jwtAuthBackend struct {
	*framework.Backend
	TokenController * backend.Backend
}

func GetBackend(t *testing.T, now func() time.Time) (*jwtAuthBackend, logical.Storage, *sharedio.MemoryStore) {
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

	b := new(jwtAuthBackend)

	mb, err := kafka.NewMessageBroker(context.TODO(), config.StorageView)
	require.NoError(t, err)
	schema, err := model.GetSchema(false)
	require.NoError(t, err)
	storage, err := sharedio.NewMemoryStore(schema, mb)

	b.TokenController = backend.NewBackend(storage, func() (string, error) {
		return "id", nil
	}, log.NewNullLogger(), now)

	b.Backend = &framework.Backend{
		BackendType:  logical.TypeCredential,
		PathsSpecial: &logical.Paths{},
		Paths: framework.PathAppend(
			[]*framework.Path{
				backend.PathEnable(b.TokenController),
				backend.PathDisable(b.TokenController),
				backend.PathConfigure(b.TokenController),
				backend.PathJWKS(b.TokenController),
				backend.PathRotateKey(b.TokenController),
			},
		),
		PeriodicFunc: b.TokenController.OnPeriodic,
	}


	err = b.Setup(context.Background(), config)
	require.NoError(t, err)

	return b, config.StorageView, storage
}

func EnableJWT(t *testing.T, b logical.Backend, storage logical.Storage) {
	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "jwt/enable",
		Storage:   storage,
		Data:      nil,
	}
	resp, err := b.HandleRequest(context.Background(), req)
	RequireValidResponse(t, resp, err)
}

func RequireValidResponse(t *testing.T, resp *logical.Response, err error) {
	require.NoError(t, err, "error on request")
	require.NotNil(t, resp, "empty response")
	require.False(t, resp.IsError(), "error response")
}

