package framework

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_servers/backend"
	"github.com/hashicorp/vault/api"
	vaulthttp "github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/vault"
	"github.com/stretchr/testify/require"
)

const (
	rootToken = "token"
)

type TestVaultCluster struct {
	t           *testing.T
	Client      *api.Client
	cleanupFunc func()
}

func NewTestVaultCluster(t *testing.T, logicalBackendsMap map[string]logical.Factory) *TestVaultCluster {
	t.Helper()

	cluster := vault.NewTestCluster(t, &vault.CoreConfig{
		DevToken:        rootToken,
		LogicalBackends: logicalBackendsMap,
	}, &vault.TestClusterOptions{HandlerFunc: vaulthttp.Handler})
	cluster.Start()
	cleanupFunc := cluster.Cleanup

	core := cluster.Cores[0].Core
	vault.TestWaitActive(t, core)
	cl := cluster.Cores[0].Client

	return &TestVaultCluster{
		t:           t,
		Client:      cl,
		cleanupFunc: cleanupFunc,
	}
}

func (v *TestVaultCluster) MountPlugin(name, path string, options map[string]string) {
	v.t.Helper()

	err := v.Client.Sys().Mount(backend.PluginName, &api.MountInput{
		Type:    backend.PluginName,
		Options: options,
	})
	require.Nil(v.t, err)
}

func (v *TestVaultCluster) Cleanup() {
	v.cleanupFunc()
}
