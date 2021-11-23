package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func createTeammates(t *testing.T, tx *io.MemoryStoreTxn, teammates ...model.FullTeammate) {
	userRepo := iam_repo.NewUserRepository(tx)
	teammateRepo := repo.NewTeammateRepository(tx)

	for _, teammate := range teammates {
		tmp := teammate
		tmp.Version = uuid.New()
		tmp.FullIdentifier = uuid.New()
		err := userRepo.Create(&tmp.User)
		require.NoError(t, err)
		err = teammateRepo.Create(tmp.GetTeammate())
		require.NoError(t, err)
	}
}

func createFlantTenant(t *testing.T, tx *io.MemoryStoreTxn) {
	repo := iam_repo.NewTenantRepository(tx)
	err := repo.Create(&iam_model.Tenant{
		UUID:       fixtures.FlantUUID,
		Version:    "v1",
		Identifier: "flant",
		Origin:     "test",
	})
	require.NoError(t, err)
}

func teammateFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	createFlantTenant(t, tx)
	createTeammates(t, tx, fixtures.Teammates()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_TeammateList(t *testing.T) {
	tx := runFixtures(t, teamFixture, teammateFixture).Txn(true)
	repo := repo.NewTeammateRepository(tx)

	teammates, err := repo.List(fixtures.TeamUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range teammates {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{
		fixtures.TeammateUUID1, fixtures.TeammateUUID3,
	}, ids)
}
