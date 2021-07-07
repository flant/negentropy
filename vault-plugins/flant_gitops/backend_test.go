package flant_gitops

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

func getTestBackend(t *testing.T, ctx context.Context) (*backend, logical.Storage, *util.TestLogger) {
	defaultLeaseTTLVal := time.Hour * 12
	maxLeaseTTLVal := time.Hour * 24

	logical.TestBackendConfig()

	testLogger := util.NewTestLogger()

	config := &logical.BackendConfig{
		Logger: testLogger.VaultLogger,
		System: &logical.StaticSystemView{
			DefaultLeaseTTLVal: defaultLeaseTTLVal,
			MaxLeaseTTLVal:     maxLeaseTTLVal,
		},
		StorageView: &logical.InmemStorage{},
	}

	b, err := newBackend()
	if err != nil {
		t.Fatalf("unable to create backend: %s", err)
	}

	if err := b.Setup(ctx, config); err != nil {
		t.Fatalf("unable to setup backend: %s", err)
	}

	return b, config.StorageView, testLogger
}
