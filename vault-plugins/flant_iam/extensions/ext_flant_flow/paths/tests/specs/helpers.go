package specs

import (
	"encoding/json"
	"net/url"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	iam_specs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	ext_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func CreateRandomClient(clientsAPI tests.TestAPI, adminUUID model.UserUUID) ext_model.Client {
	createPayload := fixtures.RandomClientCreatePayload()
	createPayload["primary_administrators"] = []string{adminUUID}
	createdData := clientsAPI.Create(tests.Params{}, nil, createPayload)
	rawClient := createdData.Get("client")
	data := []byte(rawClient.String())
	var client ext_model.Client
	err := json.Unmarshal(data, &client)
	Expect(err).ToNot(HaveOccurred())
	return client
}

func CreateDevopsTeam(teamAPI tests.TestAPI) ext_model.Team {
	createPayload := fixtures.TeamCreatePayload(fixtures.Teams()[0])
	createdData := teamAPI.Create(tests.Params{}, nil, createPayload)
	return buildGroup(createdData)
}

func CreateRandomTeam(teamAPI tests.TestAPI) ext_model.Team {
	createPayload := fixtures.RandomTeamCreatePayload()
	createdData := teamAPI.Create(tests.Params{}, nil, createPayload)
	return buildGroup(createdData)
}

func CreateRandomTeamWithParent(teamAPI tests.TestAPI, parentTeamUUID ext_model.TeamUUID) ext_model.Team {
	createPayload := fixtures.RandomTeamCreatePayload()
	createPayload["parent_team_uuid"] = parentTeamUUID
	createdData := teamAPI.Create(tests.Params{}, nil, createPayload)
	return buildGroup(createdData)
}

func CreateRandomTeamWithSpecificType(teamAPI tests.TestAPI, teamType string) ext_model.Team {
	createPayload := fixtures.RandomTeamCreatePayload()
	createPayload["team_type"] = teamType
	createdData := teamAPI.Create(tests.Params{}, nil, createPayload)
	return buildGroup(createdData)
}

func buildGroup(groupData gjson.Result) ext_model.Team {
	rawTeam := groupData.Get("team")
	data := []byte(rawTeam.String())
	var team ext_model.Team
	err := json.Unmarshal(data, &team)
	Expect(err).ToNot(HaveOccurred())
	return team
}

func CreateRandomTeammate(teamateAPI tests.TestAPI, team ext_model.Team) ext_model.FullTeammate {
	createPayload := fixtures.RandomTeammateCreatePayload(team)
	createdData := teamateAPI.Create(tests.Params{
		"team": team.UUID,
	}, nil, createPayload)
	rawTeamate := createdData.Get("teammate")
	data := []byte(rawTeamate.String())
	var teammate ext_model.FullTeammate
	err := json.Unmarshal(data, &teammate)
	Expect(err).ToNot(HaveOccurred())
	return teammate
}

func CreateRandomProject(projectAPI tests.TestAPI, clientID ext_model.ClientUUID) ext_model.Project {
	createPayload := fixtures.RandomProjectCreatePayload()
	project, err := СreateProject(projectAPI, clientID, createPayload, false)
	Expect(err).ToNot(HaveOccurred())
	return *project
}

func СreateProject(projectAPI tests.TestAPI, clientID ext_model.ClientUUID,
	createPayload map[string]interface{}, privileged bool) (*ext_model.Project, error) {
	createPayload["tenant_uuid"] = clientID
	params := tests.Params{
		"client": clientID,
	}
	var createData gjson.Result
	if privileged {
		createData = projectAPI.CreatePrivileged(params, url.Values{}, createPayload)
	} else {
		createData = projectAPI.Create(params, url.Values{}, createPayload)
	}
	rawProject := createData.Get("project")
	data := []byte(rawProject.String())
	var project ext_model.Project
	err := json.Unmarshal(data, &project)
	if err != nil {
		return nil, err
	}
	return &project, err
}

// TryCreateProjects creates projects, does not stop after error, as can be collision by uuid
func TryCreateProjects(projectAPI tests.TestAPI, clientID ext_model.ClientUUID, projects ...ext_model.Project) {
	for _, project := range projects {
		payload := fixtures.ProjectCreatePayload(project)
		_, err := СreateProject(projectAPI, clientID, payload, true) //nolint:errcheck
		Expect(err).ToNot(HaveOccurred())
	}
}

func CreateRandomContact(contactAPI tests.TestAPI, clientID ext_model.TeamUUID) ext_model.FullContact {
	createPayload := fixtures.RandomContactCreatePayload()
	createPayload["tenant_uuid"] = clientID
	createdData := contactAPI.Create(tests.Params{
		"client": clientID,
	}, nil, createPayload)
	rawContact := createdData.Get("contact")
	data := []byte(rawContact.String())
	var contact ext_model.FullContact
	err := json.Unmarshal(data, &contact)
	Expect(err).ToNot(HaveOccurred())
	return contact
}

func ConfigureFlantFlow(roleApi tests.TestAPI, teamAPI tests.TestAPI, configAPI testapi.ConfigAPI) *config.FlantFlowConfig {
	cfg := BaseConfigureFlantFlow(roleApi, configAPI)
	teamL1 := CreateRandomTeam(teamAPI)
	teamMk8s := CreateRandomTeam(teamAPI)
	teamOkmeter := CreateRandomTeam(teamAPI)
	teams := map[string]string{
		config.L1:      teamL1.UUID,
		config.Mk8s:    teamMk8s.UUID,
		config.Okmeter: teamOkmeter.UUID,
	}
	configAPI.ConfigureExtensionFlantFlowSpecificTeams(teams)

	*cfg = configAPI.ReadConfigFlantFlow()
	return cfg
}

func BaseConfigureFlantFlow(roleAPI tests.TestAPI, configAPI testapi.ConfigAPI) *config.FlantFlowConfig {
	configAPI.ConfigureExtensionFlantFlowFlantTenantUUID(uuid.New())
	r1 := iam_specs.CreateRandomRole(roleAPI)
	r2 := iam_specs.CreateRandomRole(roleAPI)
	servicePackSpecification := config.ServicePacksRolesSpecification{
		ext_model.DevOps: {
			ext_usecase.DirectMembersGroupType: []model.BoundRole{{
				Name:    r1.Name,
				Options: map[string]interface{}{"max_ttl": "1600m", "ttl": "800m"},
			}},
			ext_usecase.ManagersGroupType: []model.BoundRole{{
				Name:    r2.Name,
				Options: nil,
			}},
		},
	}
	configAPI.ConfigureExtensionServicePacksRolesSpecification(servicePackSpecification) // TODO fill later
	configAPI.ConfigureExtensionFlantFlowAllFlantGroupUUID(uuid.New())
	globalRole := iam_specs.CreateRandomRole(roleAPI)
	configAPI.ConfigureExtensionFlantFlowAllFlantGroupRoles([]string{globalRole.Name})
	primaryClientAdminRole := iam_specs.CreateRandomRole(roleAPI)
	configAPI.ConfigureExtensionFlantFlowClientPrimaryAdminsRoles([]string{primaryClientAdminRole.Name})
	cfg := configAPI.ReadConfigFlantFlow()
	return &cfg
}

func CheckGroupHasUser(groupAPI tests.TestAPI, tenantUUID model.TenantUUID, groupUUID model.GroupUUID,
	userUUID model.UserUUID, expectHas bool) {
	groupAPI.Read(tests.Params{
		"tenant": tenantUUID,
		"group":  groupUUID,
		"expectPayload": func(json gjson.Result) {
			groupData := json.Get("group")
			Expect(groupData.Map()).To(HaveKey("uuid"))
			Expect(groupData.Map()).To(HaveKey("members"))
			usersDataArray := groupData.Get("members").Array()
			var found bool
			for _, memberData := range usersDataArray {
				Expect(memberData.Map()).To(HaveKey("type"))
				Expect(memberData.Map()).To(HaveKey("uuid"))
				if memberData.Get("type").String() == "user" &&
					memberData.Get("uuid").String() == userUUID {
					found = true
					break
				}
			}
			if expectHas {
				Expect(found).To(BeTrue())
			} else {
				Expect(found).ToNot(BeTrue())
			}
		},
	},
		nil)
}
