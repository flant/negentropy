package usecase

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
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
	tx := runFixtures(t, clientFixture, projectFixture).Txn(true)
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

func Test_makeProjectCastingThroughBytes(t *testing.T) {
	project := &model.Project{
		Project: iam.Project{
			ArchiveMark: memdb.ArchiveMark{
				Timestamp: 99,
				Hash:      999,
			},
			UUID:         "u1",
			TenantUUID:   "tuid1",
			Version:      "v1",
			Identifier:   "i1",
			FeatureFlags: []iam.FeatureFlagName{"f1"},
			Extensions:   nil,
		},
		ServicePacks: map[model.ServicePackName]string{
			"DEVOPS": "some_options",
		},
	}
	iamProject, err := makeIamProject(project)
	require.NoError(t, err)
	fmt.Printf("%#v\n", iamProject)
	bytes, err := json.Marshal(iamProject)
	require.NoError(t, err)
	var newIamProject iam.Project

	err = json.Unmarshal(bytes, &newIamProject)
	require.NoError(t, err)
	fmt.Printf("%#v\n", newIamProject)

	newProject, err := makeProject(&newIamProject)
	require.NoError(t, err)
	require.Equal(t, project, newProject)
}

func Test_makeProjectDirectCasting(t *testing.T) {
	project := &model.Project{
		Project: iam.Project{
			ArchiveMark: memdb.ArchiveMark{
				Timestamp: 99,
				Hash:      999,
			},
			UUID:         "u1",
			TenantUUID:   "tuid1",
			Version:      "v1",
			Identifier:   "i1",
			FeatureFlags: []iam.FeatureFlagName{"f1"},
			Extensions:   nil,
		},
		ServicePacks: map[model.ServicePackName]string{
			"DEVOPS": "some_options",
		},
	}
	iamProject, err := makeIamProject(project)
	require.NoError(t, err)
	fmt.Printf("%#v\n", iamProject)
	newProject, err := makeProject(iamProject)
	require.NoError(t, err)
	require.Equal(t, project, newProject)
}

func Test_makeProjectDirectCastingEmptyServicePack(t *testing.T) {
	project := &model.Project{
		Project: iam.Project{
			ArchiveMark: memdb.ArchiveMark{
				Timestamp: 99,
				Hash:      999,
			},
			UUID:         "u1",
			TenantUUID:   "tuid1",
			Version:      "v1",
			Identifier:   "i1",
			FeatureFlags: []iam.FeatureFlagName{"f1"},
			Extensions:   nil,
		},
		ServicePacks: nil,
	}
	iamProject, err := makeIamProject(project)
	require.NoError(t, err)
	fmt.Printf("%#v\n", iamProject)
	newProject, err := makeProject(iamProject)
	require.NoError(t, err)
	require.Equal(t, project, newProject)
}
