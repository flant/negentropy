package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

var (
	cache = Cache{
		ServerList: ServerList{
			Tenants:  map[iam.TenantUUID]iam.Tenant{},
			Projects: map[iam.ProjectUUID]iam.Project{},
			Servers:  map[ext.ServerUUID]ext.Server{},
		},
		TenantsTimestamps:  map[iam.TenantUUID]time.Time{},
		ProjectsTimestamps: map[iam.ProjectUUID]time.Time{},
		ServersTimestamps:  map[ext.ServerUUID]time.Time{},
		TTL:                time.Second * 5,
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
	require.NotEmpty(t, c.TenantsTimestamps)
	require.NotEmpty(t, c.ProjectsTimestamps)
	require.NotEmpty(t, c.ServersTimestamps)

	c.TTL = 0
	c.ClearOverdue()

	require.Empty(t, c.Tenants)
	require.Empty(t, c.TenantsTimestamps)
	require.Empty(t, c.Projects)
	require.Empty(t, c.ProjectsTimestamps)
	require.Empty(t, c.Servers)
	require.Empty(t, c.ServersTimestamps)
}
