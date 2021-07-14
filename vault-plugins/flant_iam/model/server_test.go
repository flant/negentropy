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
