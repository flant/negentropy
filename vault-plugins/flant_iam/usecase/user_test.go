package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func Test_UserList(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture).Txn(true)
	repo := iam_repo.NewUserRepository(tx)

	users, err := repo.List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range users {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.UserUUID1, fixtures.UserUUID2, fixtures.UserUUID3, fixtures.UserUUID4}, ids)
}

func Test_forbidDoublingEmailAtTenant(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, UserFixture).Txn(true)
	repo := iam_repo.NewUserRepository(tx)
	oldUser := fixtures.Users()[0]
	extraUserWithSameEmail := &model.User{
		UUID:           uuid.New(),
		TenantUUID:     oldUser.TenantUUID,
		Identifier:     uuid.New(),
		FullIdentifier: uuid.New(),
		Email:          oldUser.Email,
		Origin:         "test",
		Version:        uuid.New(),
	}

	err := repo.Create(extraUserWithSameEmail)

	require.Error(t, err)
}
