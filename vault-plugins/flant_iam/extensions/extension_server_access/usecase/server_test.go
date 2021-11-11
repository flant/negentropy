package usecase

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

// no way to mock Vault storage right now, skipping
/*
func Test_Register(t *testing.T) {
	schema, err := ext_model.GetSchema()
	require.NoError(t, err)

	memdb, _ := io.NewMemoryStore(schema, nil)
	tx := memdb.Txn(true)
	defer tx.Abort()

	serverRepo := NewServerService(tx)

	tenant := GenerateTenantFixtures(t, tx)
	project := GenerateProjectFixtures(t, tx, tenant.UUID)
	_ = GenerateRoleFixtures(t, tx, iam_model.RoleScopeTenant)

	jwt, err := serverRepo.Create(context.TODO(), nil, tenant.UUID, project.UUID, "main", nil, nil, []string{"main"})
	require.NoError(t, err)
	require.NotEmpty(t, jwt)

	_ = tx.Commit()
}
*/

func Test_List(t *testing.T) {
	tenant := uuid.New()
	project := uuid.New()
	server := &ext_model.Server{
		UUID:        uuid.New(),
		TenantUUID:  tenant,
		ProjectUUID: project,
		Identifier:  "test",
	}

	memdb, _ := io.NewMemoryStore(&memdb.DBSchema{Tables: repo.ServerSchema()}, nil, hclog.NewNullLogger())
	tx := memdb.Txn(true)
	err := tx.Insert(ext_model.ServerType, server)
	require.NoError(t, err)
	_ = tx.Commit()

	tx = memdb.Txn(false)

	repo := repo.NewServerRepository(tx)

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

func GenerateTenantFixtures(t *testing.T, tx *io.MemoryStoreTxn) *iam_model.Tenant {
	t.Helper()

	tenantRepo := iam_repo.NewTenantRepository(tx)

	tenant := &iam_model.Tenant{
		UUID:       uuid.New(),
		Version:    iam_repo.NewResourceVersion(),
		Identifier: "main",
	}

	err := tenantRepo.Create(tenant)
	require.NoError(t, err)

	return tenant
}

func GenerateProjectFixtures(t *testing.T, tx *io.MemoryStoreTxn, tenantUUID string) *iam_model.Project {
	t.Helper()

	projectRepo := iam_repo.NewProjectRepository(tx)

	project := &iam_model.Project{
		UUID:       uuid.New(),
		TenantUUID: tenantUUID,
		Version:    iam_repo.NewResourceVersion(),
		Identifier: "main",
	}

	err := projectRepo.Create(project)
	require.NoError(t, err)

	return project
}

func GenerateRoleFixtures(t *testing.T, tx *io.MemoryStoreTxn, roleScope iam_model.RoleScope) *iam_model.Role {
	t.Helper()

	roleRepo := iam_repo.NewRoleRepository(tx)

	role := &iam_model.Role{Name: "main"}

	switch roleScope {
	case iam_model.RoleScopeTenant:
		role.Scope = iam_model.RoleScopeTenant
	case iam_model.RoleScopeProject:
		role.Scope = iam_model.RoleScopeProject
	}

	err := roleRepo.Create(role)
	require.NoError(t, err)

	return role
}
