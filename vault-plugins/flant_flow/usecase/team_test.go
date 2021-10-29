package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createTeams(t *testing.T, repo *iam_repo.TeamRepository, teams ...model.Team) {
	for _, team := range teams {
		tmp := team
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func teamFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewTeamRepository(tx)
	createTeams(t, repo, fixtures.Teams()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_TeamList(t *testing.T) {
	tx := runFixtures(t, teamFixture).Txn(true)
	repo := iam_repo.NewTeamRepository(tx)

	teams, err := repo.List(false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range teams {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.TeamUUID1, fixtures.TeamUUID2, fixtures.TeamUUID3}, ids)
}
