package usecase

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	projectUUID1 = "00000000-0100-0000-0000-000000000000"
	projectUUID2 = "00000000-0200-0000-0000-000000000000"
	projectUUID3 = "00000000-0300-0000-0000-000000000000"
	projectUUID4 = "00000000-0400-0000-0000-000000000000"
	projectUUID5 = "00000000-0500-0000-0000-000000000000"
)

var (
	pr1 = model.Project{
		UUID:       projectUUID1,
		TenantUUID: tenantUUID1,
		Identifier: "pr1",
	}
	pr2 = model.Project{
		UUID:       projectUUID2,
		TenantUUID: tenantUUID1,
		Identifier: "pr2",
	}
	pr3 = model.Project{
		UUID:       projectUUID3,
		TenantUUID: tenantUUID1,
		Identifier: "pr3",
	}
	pr4 = model.Project{
		UUID:       projectUUID4,
		TenantUUID: tenantUUID1,
		Identifier: "pr4",
	}
	pr5 = model.Project{
		UUID:       projectUUID5,
		TenantUUID: tenantUUID2,
		Identifier: "pr5",
	}
)

func createProjects(t *testing.T, repo *ProjectsService, projects ...model.Project) {
	for _, project := range projects {
		tmp := project
		err := repo.Create(&tmp)
		dieOnErr(t, err)
	}
}

func projectFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := Projects(tx)
	createProjects(t, repo, []model.Project{pr1, pr2, pr3, pr4, pr5}...)
	err := tx.Commit()
	dieOnErr(t, err)
}

func Test_ProjectList(t *testing.T) {
	tx := runFixtures(t, tenantFixture, projectFixture).Txn(true)

	projects, err := Projects(tx).List(tenantUUID1)

	dieOnErr(t, err)
	ids := make([]string, 0)
	for _, obj := range projects {
		ids = append(ids, obj.ObjId())
	}
	checkDeepEqual(t, []string{projectUUID1, projectUUID2, projectUUID3, projectUUID4}, ids)
}
