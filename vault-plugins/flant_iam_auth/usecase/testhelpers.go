package usecase

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func runFixtures(t *testing.T, fixtures ...func(t *testing.T, store *io.MemoryStore)) *io.MemoryStore {
	schema, err := repo.GetSchema()
	require.NoError(t, err)
	store, err := io.NewMemoryStore(schema, nil, hclog.NewNullLogger())
	require.NoError(t, err)
	for _, fixture := range fixtures {
		fixture(t, store)
	}
	return store
}

func createRoles(t *testing.T, repo *iam_repo.RoleRepository, roles ...model.Role) {
	for _, role := range roles {
		tmp := role
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func roleFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewRoleRepository(tx)
	createRoles(t, repo, fixtures.Roles()...)
	err := tx.Commit()
	require.NoError(t, err)
}
