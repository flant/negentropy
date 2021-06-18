package integration

import (
	"path"
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/flant_servers/backend"
	"github.com/flant/negentropy/vault-plugins/flant_servers/testing/integration/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerRegister(t *testing.T) {
	testVaultCluster := framework.NewTestVaultCluster(t, map[string]logical.Factory{backend.PluginName: backend.Factory})
	defer testVaultCluster.Cleanup()

	tenantUUID := uuid.New()
	projectUUID := uuid.New()

	testVaultCluster.MountPlugin(backend.PluginName, backend.PluginName, map[string]string{
		"testRun":         "",
		"testTenantUUID":  tenantUUID,
		"testProjectUUID": projectUUID,
	})

	serverRegistration := map[string]interface{}{
		"identifier": "test",
		"labels": map[string]interface{}{
			"test": "test",
		},
		"annotations": map[string]interface{}{
			"test": "test",
		},
	}

	secret, err := testVaultCluster.Client.Logical().Write(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "register_server"), serverRegistration)
	require.Nil(t, err)
	assert.Equal(t, secret.Data["identifier"], serverRegistration["identifier"])
	assert.Equal(t, secret.Data["labels"], serverRegistration["labels"])
	assert.Equal(t, secret.Data["annotations"], serverRegistration["annotations"])

	// second write with the same identifier should fail
	_, err = testVaultCluster.Client.Logical().Write(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "register_server"), serverRegistration)
	require.NotNil(t, err)
}

func TestServerUpdate(t *testing.T) {
	testVaultCluster := framework.NewTestVaultCluster(t, map[string]logical.Factory{backend.PluginName: backend.Factory})
	defer testVaultCluster.Cleanup()

	tenantUUID := uuid.New()
	projectUUID := uuid.New()

	testVaultCluster.MountPlugin(backend.PluginName, backend.PluginName, map[string]string{
		"testRun":         "",
		"testTenantUUID":  tenantUUID,
		"testProjectUUID": projectUUID,
	})

	serverRegistration := map[string]interface{}{
		"identifier": "test",
		"labels": map[string]interface{}{
			"test": "test",
		},
		"annotations": map[string]interface{}{
			"test": "test",
		},
	}

	secret, err := testVaultCluster.Client.Logical().Write(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "register_server"), serverRegistration)
	require.Nil(t, err)
	require.NotEmpty(t, secret.Data["uuid"])
	require.NotEmpty(t, secret.Data["resource_version"])
	assert.Equal(t, secret.Data["identifier"], serverRegistration["identifier"])
	assert.Equal(t, secret.Data["labels"], serverRegistration["labels"])
	assert.Equal(t, secret.Data["annotations"], serverRegistration["annotations"])

	newServer := map[string]interface{}{
		"identifier": "another",
		"labels": map[string]interface{}{
			"another": "another",
		},
		"annotations": map[string]interface{}{
			"another": "another",
		},
		"resource_version": secret.Data["resource_version"],
	}

	newServerRes, err := testVaultCluster.Client.Logical().Write(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "server", secret.Data["uuid"].(string)), newServer)
	require.Nil(t, err)
	require.NotEmpty(t, secret.Data["uuid"])
	require.NotEmpty(t, secret.Data["resource_version"])
	assert.Equal(t, newServerRes.Data["identifier"], newServer["identifier"])
	assert.Equal(t, newServerRes.Data["labels"], newServer["labels"])
	assert.Equal(t, newServerRes.Data["annotations"], newServer["annotations"])
}

func TestServerRead(t *testing.T) {
	testVaultCluster := framework.NewTestVaultCluster(t, map[string]logical.Factory{backend.PluginName: backend.Factory})
	defer testVaultCluster.Cleanup()

	tenantUUID := uuid.New()
	projectUUID := uuid.New()

	testVaultCluster.MountPlugin(backend.PluginName, backend.PluginName, map[string]string{
		"testRun":         "",
		"testTenantUUID":  tenantUUID,
		"testProjectUUID": projectUUID,
	})

	serverRegistration := map[string]interface{}{
		"identifier": "test",
		"labels": map[string]interface{}{
			"test": "test",
		},
		"annotations": map[string]interface{}{
			"test": "test",
		},
	}

	secret, err := testVaultCluster.Client.Logical().Write(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "register_server"), serverRegistration)
	require.Nil(t, err)
	require.NotEmpty(t, secret.Data["uuid"])
	require.NotEmpty(t, secret.Data["resource_version"])
	assert.Equal(t, secret.Data["identifier"], serverRegistration["identifier"])
	assert.Equal(t, secret.Data["labels"], serverRegistration["labels"])
	assert.Equal(t, secret.Data["annotations"], serverRegistration["annotations"])

	newServerRes, err := testVaultCluster.Client.Logical().Read(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "server", secret.Data["uuid"].(string)))
	require.Nil(t, err)
	require.NotEmpty(t, secret.Data["uuid"])
	require.NotEmpty(t, secret.Data["resource_version"])
	assert.Equal(t, newServerRes.Data["identifier"], serverRegistration["identifier"])
	assert.Equal(t, newServerRes.Data["labels"], serverRegistration["labels"])
	assert.Equal(t, newServerRes.Data["annotations"], serverRegistration["annotations"])
}

func TestServerList(t *testing.T) {
	testVaultCluster := framework.NewTestVaultCluster(t, map[string]logical.Factory{backend.PluginName: backend.Factory})
	defer testVaultCluster.Cleanup()

	tenantUUID := uuid.New()
	projectUUID := uuid.New()

	testVaultCluster.MountPlugin(backend.PluginName, backend.PluginName, map[string]string{
		"testRun":         "",
		"testTenantUUID":  tenantUUID,
		"testProjectUUID": projectUUID,
	})

	serverRegistration := map[string]interface{}{
		"identifier": "test",
		"labels": map[string]interface{}{
			"test": "test",
		},
		"annotations": map[string]interface{}{
			"test": "test",
		},
	}

	secret, err := testVaultCluster.Client.Logical().Write(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "register_server"), serverRegistration)
	require.Nil(t, err)
	require.NotEmpty(t, secret.Data["uuid"])
	require.NotEmpty(t, secret.Data["resource_version"])
	assert.Equal(t, secret.Data["identifier"], serverRegistration["identifier"])
	assert.Equal(t, secret.Data["labels"], serverRegistration["labels"])
	assert.Equal(t, secret.Data["annotations"], serverRegistration["annotations"])

	newServerRes, err := testVaultCluster.Client.Logical().List(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "servers"))
	require.Nil(t, err)
	require.NotEmpty(t, newServerRes.Data["uuids"])
	assert.Contains(t, newServerRes.Data["uuids"], secret.Data["uuid"])
}

func TestServerDelete(t *testing.T) {
	testVaultCluster := framework.NewTestVaultCluster(t, map[string]logical.Factory{backend.PluginName: backend.Factory})
	defer testVaultCluster.Cleanup()

	tenantUUID := uuid.New()
	projectUUID := uuid.New()

	testVaultCluster.MountPlugin(backend.PluginName, backend.PluginName, map[string]string{
		"testRun":         "",
		"testTenantUUID":  tenantUUID,
		"testProjectUUID": projectUUID,
	})

	serverRegistration := map[string]interface{}{
		"identifier": "test",
		"labels": map[string]interface{}{
			"test": "test",
		},
		"annotations": map[string]interface{}{
			"test": "test",
		},
	}

	secret, err := testVaultCluster.Client.Logical().Write(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "register_server"), serverRegistration)
	require.Nil(t, err)
	require.NotEmpty(t, secret.Data["uuid"])
	require.NotEmpty(t, secret.Data["resource_version"])
	assert.Equal(t, secret.Data["identifier"], serverRegistration["identifier"])
	assert.Equal(t, secret.Data["labels"], serverRegistration["labels"])
	assert.Equal(t, secret.Data["annotations"], serverRegistration["annotations"])

	_, err = testVaultCluster.Client.Logical().Delete(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "server", secret.Data["uuid"].(string)))
	require.Nil(t, err)

	_, err = testVaultCluster.Client.Logical().Read(path.Join(
		backend.PluginName, "tenant", tenantUUID, "project", projectUUID, "server", secret.Data["uuid"].(string)))
	require.NotNil(t, err)
}
