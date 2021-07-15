package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func Test_ServerDbSchema(t *testing.T) {
	schema := model.TenantSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("server schema is invalid: %v", err)
	}
}

func Test_Register(t *testing.T) {
	t.Skipf("importt cycle for schemas should be fixed first")

	schema, err := model.GetSchema()
	require.NoError(t, err)

	memdb, _ := io.NewMemoryStore(schema, nil)
	tx := memdb.Txn(true)
	defer tx.Abort()

	serverRepo := NewServerRepository(tx)

	tenant := GenerateTenantFixtures(t, tx)
	project := GenerateProjectFixtures(t, tx, tenant.UUID)
	_ = GenerateRoleFixtures(t, tx, model.RoleScopeTenant)

	server := &Server{
		UUID:        uuid.New(),
		TenantUUID:  tenant.UUID,
		ProjectUUID: project.UUID,
		Version:     model.NewResourceVersion(),
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

func GenerateTenantFixtures(t *testing.T, tx *io.MemoryStoreTxn) *model.Tenant {
	t.Helper()

	tenantRepo := model.NewTenantRepository(tx)

	tenant := &model.Tenant{
		UUID:       uuid.New(),
		Version:    model.NewResourceVersion(),
		Identifier: "main",
	}

	err := tenantRepo.Create(tenant)
	require.NoError(t, err)

	return tenant
}

func GenerateProjectFixtures(t *testing.T, tx *io.MemoryStoreTxn, tenantUUID string) *model.Project {
	t.Helper()

	projectRepo := model.NewProjectRepository(tx)

	project := &model.Project{
		UUID:       uuid.New(),
		TenantUUID: tenantUUID,
		Version:    model.NewResourceVersion(),
		Identifier: "main",
	}

	err := projectRepo.Create(project)
	require.NoError(t, err)

	return project
}

func GenerateRoleFixtures(t *testing.T, tx *io.MemoryStoreTxn, roleScope model.RoleScope) *model.Role {
	t.Helper()

	roleRepo := model.NewRoleRepository(tx)

	role := &model.Role{Name: "main"}

	switch roleScope {
	case model.RoleScopeTenant:
		role.Scope = model.RoleScopeTenant
	case model.RoleScopeProject:
		role.Scope = model.RoleScopeProject
	}

	err := roleRepo.Create(role)
	require.NoError(t, err)

	return role
}
