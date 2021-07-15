package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func Test_ServerDbSchema(t *testing.T) {
	schema := TenantSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("server schema is invalid: %v", err)
	}
}

func Test_Register(t *testing.T) {
	schema, err := GetSchema()
	require.NoError(t, err)

	memdb, _ := io.NewMemoryStore(schema, nil)
	tx := memdb.Txn(true)
	defer tx.Abort()

	serverRepo := NewServerRepository(tx)

	tenant := GenerateTenantFixtures(t, tx)
	project := GenerateProjectFixtures(t, tx, tenant.UUID)
	_ = GenerateRoleFixtures(t, tx, RoleScopeTenant)

	server := &Server{
		UUID:        uuid.New(),
		TenantUUID:  tenant.UUID,
		ProjectUUID: project.UUID,
		Version:     NewResourceVersion(),
		Identifier:  "main",
	}

	err = serverRepo.Create(server, []string{"main"})
	require.NoError(t, err)

	_ = tx.Commit()
}

func Test_List(t *testing.T) {
	tenant := uuid.New()
	project := uuid.New()
	server := &Server{
		UUID:        uuid.New(),
		TenantUUID:  tenant,
		ProjectUUID: project,
		Identifier:  "test",
	}

	memdb, _ := io.NewMemoryStore(ServerSchema(), nil)
	tx := memdb.Txn(true)
	err := tx.Insert(ServerType, server)
	require.NoError(t, err)
	_ = tx.Commit()

	tx = memdb.Txn(false)

	repo := NewServerRepository(tx)

	t.Run("find by tenant and project", func(t *testing.T) {
		list, err := repo.List(tenant, project)
		assert.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, server, list[0])
	})

	t.Run("find by tenant", func(t *testing.T) {
		list, err := repo.List(tenant, "")
		assert.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, server, list[0])
	})

	t.Run("find by project", func(t *testing.T) {
		list, err := repo.List("", project)
		assert.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, server, list[0])
	})

	t.Run("full scan list", func(t *testing.T) {
		list, err := repo.List("", "")
		assert.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, server, list[0])
	})
}

func GenerateTenantFixtures(t *testing.T, tx *io.MemoryStoreTxn) *Tenant {
	t.Helper()

	tenantRepo := NewTenantRepository(tx)

	tenant := &Tenant{
		UUID:       uuid.New(),
		Version:    NewResourceVersion(),
		Identifier: "main",
	}

	err := tenantRepo.Create(tenant)
	require.NoError(t, err)

	return tenant
}

func GenerateProjectFixtures(t *testing.T, tx *io.MemoryStoreTxn, tenantUUID string) *Project {
	t.Helper()

	projectRepo := NewProjectRepository(tx)

	project := &Project{
		UUID:       uuid.New(),
		TenantUUID: tenantUUID,
		Version:    NewResourceVersion(),
		Identifier: "main",
	}

	err := projectRepo.Create(project)
	require.NoError(t, err)

	return project
}

func GenerateRoleFixtures(t *testing.T, tx *io.MemoryStoreTxn, roleScope RoleScope) *Role {
	t.Helper()

	roleRepo := NewRoleRepository(tx)

	role := &Role{Name: "main"}

	switch roleScope {
	case RoleScopeTenant:
		role.Scope = RoleScopeTenant
	case RoleScopeProject:
		role.Scope = RoleScopeProject
	}

	err := roleRepo.Create(role)
	require.NoError(t, err)

	return role
}
