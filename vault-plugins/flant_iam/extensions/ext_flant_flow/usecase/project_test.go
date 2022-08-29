package usecase

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

var cfg = config.FlantFlowConfig{
	FlantTenantUUID: fixtures.TenantUUID1,
	SpecificTeams:   nil,
	ServicePacksRolesSpecification: config.ServicePacksRolesSpecification{
		model.DevOps: {
			DirectMembersGroupType: []iam.BoundRole{{
				Name:    "ssh",
				Options: map[string]interface{}{"max_ttl": "1600m", "ttl": "800m"},
			}},
			ManagersGroupType: []iam.BoundRole{{
				Name:    "flant.client.manage",
				Options: nil,
			}},
		},
	},
}

func createProjects(t *testing.T, srv *ProjectService, projects ...model.Project) {
	for _, project := range projects {
		sps := map[model.ServicePackName]struct{}{}
		for spn := range project.ServicePacks {
			sps[spn] = struct{}{}
		}

		iamProject, _ := makeIamProject(&project)
		tmp := ProjectParams{
			IamProject:       iamProject,
			ServicePackNames: sps,
			DevopsTeamUUID:   fixtures.TeamUUID1,
		}

		_, err := srv.Create(tmp)
		require.NoError(t, err)
	}
}

func projectFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	srv := Projects(tx, &cfg)
	createProjects(t, srv, fixtures.Projects()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_ProjectList(t *testing.T) {
	tx := runFixtures(t, teamFixture, clientFixture, projectFixture).Txn(true)
	projects, err := Projects(tx, &cfg).List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range projects {
		ids = append(ids, obj.UUID)
	}
	require.ElementsMatch(t, []string{
		fixtures.ProjectUUID1, fixtures.ProjectUUID2, fixtures.ProjectUUID3, fixtures.ProjectUUID4,
	}, ids)
}

func Test_makeProjectCastingThroughBytes(t *testing.T) {
	project := &model.Project{
		ArchiveMark: memdb.ArchiveMark{
			Timestamp: 99,
			Hash:      999,
		},
		UUID:         "u1",
		TenantUUID:   "tuid1",
		Version:      "v1",
		Identifier:   "i1",
		FeatureFlags: []iam.FeatureFlagName{"f1"},
		ServicePacks: map[model.ServicePackName]model.ServicePackCFG{
			model.DevOps: model.DevopsServicePackCFG{
				Team: "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
			},
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
		ArchiveMark: memdb.ArchiveMark{
			Timestamp: 99,
			Hash:      999,
		},
		UUID:         "u1",
		TenantUUID:   "tuid1",
		Version:      "v1",
		Identifier:   "i1",
		FeatureFlags: []iam.FeatureFlagName{"f1"},
		ServicePacks: map[model.ServicePackName]model.ServicePackCFG{
			model.DevOps: model.DevopsServicePackCFG{
				Team: "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
			},
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
		ArchiveMark: memdb.ArchiveMark{
			Timestamp: 99,
			Hash:      999,
		},
		UUID:         "u1",
		TenantUUID:   "tuid1",
		Version:      "v1",
		Identifier:   "i1",
		FeatureFlags: []iam.FeatureFlagName{"f1"},
		ServicePacks: nil,
	}
	iamProject, err := makeIamProject(project)
	require.NoError(t, err)
	fmt.Printf("%#v\n", iamProject)
	newProject, err := makeProject(iamProject)
	require.NoError(t, err)
	require.Equal(t, project, newProject)
}
