package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func createTeammates(t *testing.T, repo *iam_repo.TeammateRepository, teammates ...model.Teammate) {
	for _, teammate := range teammates {
		tmp := teammate
		tmp.Version = uuid.New()
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func teammateFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewTeammateRepository(tx)
	createTeammates(t, repo, fixtures.Teammates()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_TeammateList(t *testing.T) {
	tx := runFixtures(t, teammateFixture).Txn(true)
	repo := iam_repo.NewTeammateRepository(tx)

	teammates, err := repo.List(false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range teammates {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{
		fixtures.TeammateUUID1, fixtures.TeammateUUID2, fixtures.TeammateUUID3,
		fixtures.TeammateUUID4, fixtures.TeammateUUID5,
	}, ids)
}
