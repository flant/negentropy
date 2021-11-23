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

func createContacts(t *testing.T, tx *io.MemoryStoreTxn, client model.Client, contacts ...model.FullContact) {
	err := iam_repo.NewTenantRepository(tx).Create(&client)
	require.NoError(t, err)
	userRepo := iam_repo.NewUserRepository(tx)
	contactRepo := repo.NewContactRepository(tx)

	for _, contact := range contacts {
		tmp := contact
		tmp.Version = uuid.New()
		tmp.FullIdentifier = uuid.New()
		err = userRepo.Create(&tmp.User)
		require.NoError(t, err)
		err = contactRepo.Create(tmp.GetContact())
		require.NoError(t, err)
	}
}

func contactFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	createContacts(t, tx, iam_model.Tenant{
		UUID:       uuid.New(),
		Version:    uuid.New(),
		Identifier: uuid.New(),
		Origin:     "test",
	}, fixtures.Contacts()...)

	err := tx.Commit()
	require.NoError(t, err)
}

func Test_ContactList(t *testing.T) {
	tx := runFixtures(t, clientFixture, contactFixture).Txn(true)
	repo := repo.NewContactRepository(tx)

	contacts, err := repo.List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range contacts {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.UserUUID1, fixtures.UserUUID2, fixtures.UserUUID3, fixtures.UserUUID4}, ids)
}
