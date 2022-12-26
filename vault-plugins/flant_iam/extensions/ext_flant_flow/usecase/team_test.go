package usecase

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func createTeams(t *testing.T, service *TeamService, teams ...model.Team) {
	for _, team := range teams {
		tmp := team
		err := service.Create(&tmp)
		require.NoError(t, err)
	}
}

func teamFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	err := iam_repo.NewTenantRepository(tx).Create(&iam_model.Tenant{
		ArchiveMark: memdb.ArchiveMark{},
		UUID:        fixtures.FlantUUID,
		Version:     "new_version",
		Identifier:  "flant",
	})
	require.NoError(t, err)
	teamService := Teams(tx, &config.FlantFlowConfig{
		FlantTenantUUID: fixtures.FlantUUID,
	})
	createTeams(t, teamService, fixtures.Teams()...)
	err = tx.Commit()
	require.NoError(t, err)
}

func Test_TeamList(t *testing.T) {
	tx := runFixtures(t, teamFixture).Txn(true)
	repo := repo.NewTeamRepository(tx)

	teams, err := repo.List(false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range teams {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.TeamUUID1, fixtures.TeamUUID2, fixtures.TeamUUID3}, ids)
}

func Test_TeamEraseAfterDelete(t *testing.T) {
	tx := runFixtures(t, teamFixture).Txn(true)
	teamService := Teams(tx, &config.FlantFlowConfig{
		FlantTenantUUID: fixtures.FlantUUID,
	})
	team1, err := teamService.GetByID(fixtures.TeamUUID1)
	require.NoError(t, err)
	linkedGroups := team1.Groups
	err = teamService.Delete(fixtures.TeamUUID1)
	require.NoError(t, err)
	// ====
	err = teamService.Erase(fixtures.TeamUUID1)
	require.NoError(t, err)
	// ====
	// Checks team and groups are erased
	team, err := repo.NewTeamRepository(tx).GetByID(fixtures.TeamUUID1)
	require.ErrorIs(t, err, consts.ErrNotFound)
	require.Nil(t, team)
	for _, lg := range linkedGroups {
		group, err := iam_repo.NewGroupRepository(tx).GetByID(lg.GroupUUID)
		require.ErrorIs(t, err, consts.ErrNotFound, fmt.Sprintf("checking is deleted %v", lg))
		require.Nil(t, group, fmt.Sprintf("checking is deleted %v", lg))
	}
}
