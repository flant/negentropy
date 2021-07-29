package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createProjects(t *testing.T, repo *ProjectService, projects ...model.Project) {
	for _, project := range projects {
		tmp := project
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func projectFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := Projects(tx)
	createProjects(t, repo, fixtures.Projects()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_ProjectList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, projectFixture).Txn(true)

	projects, err := Projects(tx).List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range projects {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{
		fixtures.ProjectUUID1, fixtures.ProjectUUID2, fixtures.ProjectUUID3, fixtures.ProjectUUID4,
	}, ids)
}
