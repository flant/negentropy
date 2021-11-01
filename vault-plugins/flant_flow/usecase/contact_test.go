package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func createContacts(t *testing.T, repo *repo.ContactRepository, contacts ...model.Contact) {
	for _, contact := range contacts {
		tmp := contact
		tmp.Version = uuid.New()
		tmp.FullIdentifier = uuid.New()
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func contactFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := repo.NewContactRepository(tx)
	createContacts(t, repo, fixtures.Contacts()...)
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
