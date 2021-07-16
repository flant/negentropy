package test

import (
	"context"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/logging"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func GetStorage(t *testing.T, config *logical.BackendConfig) *sharedio.MemoryStore{
	mb, err := kafka.NewMessageBroker(context.TODO(), config.StorageView)
	require.NoError(t, err)
	schema, err := model.GetSchema(false)
	require.NoError(t, err)
	storage, err := sharedio.NewMemoryStore(schema, mb)

	return storage
}

func PrepareBackend(t *testing.T) *logical.BackendConfig {
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

	return config
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

