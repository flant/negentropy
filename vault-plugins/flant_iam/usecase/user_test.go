package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createUsers(t *testing.T, repo *iam_repo.UserRepository, users ...model.User) {
	for _, user := range users {
		tmp := user
		tmp.Version = uuid.New()
		tmp.FullIdentifier = uuid.New()
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func userFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewUserRepository(tx)
	createUsers(t, repo, fixtures.Users()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_UserList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture).Txn(true)
	repo := iam_repo.NewUserRepository(tx)

	users, err := repo.List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range users {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.UserUUID1, fixtures.UserUUID2, fixtures.UserUUID3, fixtures.UserUUID4}, ids)
}
