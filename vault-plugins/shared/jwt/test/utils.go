package test

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"

	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func GetStorage(t *testing.T, config *logical.BackendConfig) *sharedio.MemoryStore {
	mb, err := kafka.NewMessageBroker(context.TODO(), config.StorageView, hclog.NewNullLogger())
	require.NoError(t, err)
	schema, err := model.GetSchema(false)
	require.NoError(t, err)
	storage, err := sharedio.NewMemoryStore(schema, mb, hclog.NewNullLogger())

	return storage
}

func PrepareBackend(t *testing.T) *logical.BackendConfig {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	config := &logical.BackendConfig{
		Logger: logging.NewVaultLogger(hclog.Trace),

		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}

	return config
}

func EnableJWT(t *testing.T, b logical.Backend, storage logical.Storage, isEnable bool) {
	u := "jwt/enable"
	if !isEnable {
		u = "jwt/disable"
	}

	req := &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      u,
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

func GetNowFn(sec int64) func() time.Time {
	return func() time.Time {
		return time.Unix(sec, 0)
	}
}
