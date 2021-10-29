package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_flow/iam_clients"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
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
	pc, _ := iam_clients.NewProjectClient()
	tx := store.Txn(true)
	repo := Projects(tx, pc)
	createProjects(t, repo, fixtures.Projects()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_ProjectList(t *testing.T) {
	tx := runFixtures(t, clientFixture, projectFixture).Txn(true)
	pc, _ := iam_clients.NewProjectClient()
	projects, err := Projects(tx, pc).List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range projects {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{
		fixtures.ProjectUUID1, fixtures.ProjectUUID2, fixtures.ProjectUUID3, fixtures.ProjectUUID4,
	}, ids)
}
