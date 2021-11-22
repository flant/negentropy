package model

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

const testPath = "test.json"

var (
	cache = Cache{
		ServerList: ServerList{
			Tenants:  map[iam.TenantUUID]iam.Tenant{},
			Projects: map[iam.ProjectUUID]iam.Project{},
			Servers:  map[ext.ServerUUID]ext.Server{},
		},
		tenantsTimestamps:  map[iam.TenantUUID]time.Time{},
		projectsTimestamps: map[iam.ProjectUUID]time.Time{},
		serversTimestamps:  map[ext.ServerUUID]time.Time{},
		ttl:                time.Second * 5,
	}
	list = ServerList{
		Tenants: map[iam.TenantUUID]iam.Tenant{"tu1": {
			UUID:       "tu1",
			Version:    "v1",
			Identifier: "t1",
		}},
		Projects: map[iam.ProjectUUID]iam.Project{"pu1": {
			UUID:       "pu1",
			TenantUUID: "tu1",
			Version:    "v1",
			Identifier: "p1",
		}},
		Servers: map[ext.ServerUUID]ext.Server{"su1": {
			UUID:        "su1",
			TenantUUID:  "tu1",
			ProjectUUID: "pu1",
			Version:     "v1",
			Identifier:  "s1",
		}},
	}
)

func Test_updateCache(t *testing.T) {
	c := cache

	c.Update(list)

	require.Equal(t, c.Tenants, list.Tenants)
	require.Equal(t, c.Projects, list.Projects)
	require.Equal(t, c.Servers, list.Servers)
}

func Test_clearOverdue(t *testing.T) {
	c := cache
	c.Update(list)
	require.NotEmpty(t, c.Tenants, list.Tenants)
	require.NotEmpty(t, c.Projects, list.Projects)
	require.NotEmpty(t, c.Servers, list.Servers)
	require.NotEmpty(t, c.tenantsTimestamps)
	require.NotEmpty(t, c.projectsTimestamps)
	require.NotEmpty(t, c.serversTimestamps)

	c.ttl = 0
	c.ClearOverdue()

	require.Empty(t, c.Tenants)
	require.Empty(t, c.tenantsTimestamps)
	require.Empty(t, c.Projects)
	require.Empty(t, c.projectsTimestamps)
	require.Empty(t, c.Servers)
	require.Empty(t, c.serversTimestamps)
}

func Test_SaveToFile(t *testing.T) {
	err := deleteFileIfExists(testPath)
	require.NoError(t, err)
	c := cache
	c.Update(list)
	require.NotEmpty(t, c.Tenants, list.Tenants)
	require.NotEmpty(t, c.Projects, list.Projects)
	require.NotEmpty(t, c.Servers, list.Servers)
	require.NotEmpty(t, c.tenantsTimestamps)
	require.NotEmpty(t, c.projectsTimestamps)
	require.NotEmpty(t, c.serversTimestamps)

	err = c.SaveToFile(testPath)

	require.NoError(t, err)
	require.FileExists(t, testPath)
	deleteFileIfExists(testPath)
}

func Test_ReadFromFile(t *testing.T) {
	err := deleteFileIfExists(testPath)
	require.NoError(t, err)
	c := cache
	c.Update(list)
	require.NotEmpty(t, c.Tenants, list.Tenants)
	require.NotEmpty(t, c.Projects, list.Projects)
	require.NotEmpty(t, c.Servers, list.Servers)
	require.NotEmpty(t, c.tenantsTimestamps)
	require.NotEmpty(t, c.projectsTimestamps)
	require.NotEmpty(t, c.serversTimestamps)
	err = c.SaveToFile(testPath)
	require.NoError(t, err)
	require.FileExists(t, testPath)

	c2, err := ReadFromFile(testPath, time.Second*6)

	require.NoError(t, err)
	require.Equal(t, c2.Tenants, list.Tenants)
	require.Equal(t, c2.Projects, list.Projects)
	require.Equal(t, c2.Servers, list.Servers)
	require.Equal(t, c2.ttl, time.Second*6)
	deleteFileIfExists(testPath)
}

func deleteFileIfExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

func Test_TryReadCacheFromFilePermissions(t *testing.T) {
	// No preparations

	_, err := TryReadCacheFromFile("./A/B/test.json", 100*time.Millisecond)

	assert.NoError(t, err)
	fi, err := os.Stat("./A/B")
	assert.NoError(t, err)
	perm := fi.Mode().String()
	assert.Equal(t, "drwx------", perm)
	fi, err = os.Stat("./A")
	assert.NoError(t, err)
	perm = fi.Mode().String()
	assert.Equal(t, "drwx------", perm)
	os.RemoveAll("./A/B")
}
