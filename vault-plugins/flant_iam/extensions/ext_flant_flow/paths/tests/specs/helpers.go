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
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
)

func CreateRandomClient(clientsAPI testapi.TestAPI) model.Client {
	createPayload := fixtures.RandomClientCreatePayload()
	var createdData gjson.Result
	clientsAPI.Create(testapi.Params{
		"expectPayload": func(json gjson.Result) {
			createdData = json
		},
	}, nil, createPayload)
	rawClient := createdData.Get("client")
	data := []byte(rawClient.String())
	var client model.Client
	err := json.Unmarshal(data, &client)
	Expect(err).ToNot(HaveOccurred())
	return client
}

func CreateRandomTeam(teamAPI testapi.TestAPI) model.Team {
	createPayload := fixtures.RandomTeamCreatePayload()
	var createdData gjson.Result
	teamAPI.Create(testapi.Params{
		"expectPayload": func(json gjson.Result) {
			createdData = json
		},
	}, nil, createPayload)
	rawTeam := createdData.Get("team")
	data := []byte(rawTeam.String())
	var team model.Team
	err := json.Unmarshal(data, &team)
	Expect(err).ToNot(HaveOccurred())
	return team
}

func CreateRandomTeammate(teamateAPI testapi.TestAPI, team model.Team) model.FullTeammate {
	createPayload := fixtures.RandomTeammateCreatePayload(team)
	createdData := teamateAPI.Create(testapi.Params{
		"team": team.UUID,
	}, nil, createPayload)
	rawTeamate := createdData.Get("teammate")
	data := []byte(rawTeamate.String())
	var teammate model.FullTeammate
	err := json.Unmarshal(data, &teammate)
	Expect(err).ToNot(HaveOccurred())
	return teammate
}

func CreateRandomProject(projectAPI testapi.TestAPI, clientID model.ClientUUID) model.Project {
	createPayload := fixtures.RandomProjectCreatePayload()
	project, err := createProject(projectAPI, clientID, createPayload, false)
	Expect(err).ToNot(HaveOccurred())
	return *project
}

func createProject(projectAPI testapi.TestAPI, clientID model.ClientUUID,
	createPayload map[string]interface{}, privileged bool) (*model.Project, error) {
	createPayload["tenant_uuid"] = clientID
	params := testapi.Params{
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
	var project model.Project
	err := json.Unmarshal(data, &project)
	if err != nil {
		return nil, err
	}
	return &project, err
}

// TryCreateProjects creates projects, does not stop after error, as can be collision by uuid
func TryCreateProjects(projectAPI testapi.TestAPI, clientID model.ClientUUID, projects ...model.Project) {
	for _, project := range projects {
		bytes, _ := json.Marshal(project)
		var payload map[string]interface{}
		json.Unmarshal(bytes, &payload)                              //nolint:errcheck
		_, err := createProject(projectAPI, clientID, payload, true) //nolint:errcheck
		Expect(err).ToNot(HaveOccurred())
	}
}

func CreateRandomContact(contactAPI testapi.TestAPI, clientID model.TeamUUID) model.FullContact {
	createPayload := fixtures.RandomContactCreatePayload()
	createPayload["tenant_uuid"] = clientID
	createdData := contactAPI.Create(testapi.Params{
		"client": clientID,
	}, nil, createPayload)
	rawContact := createdData.Get("contact")
	data := []byte(rawContact.String())
	var contact model.FullContact
	err := json.Unmarshal(data, &contact)
	Expect(err).ToNot(HaveOccurred())
	return contact
}

func ConfigureFlantFlow(tenantAPI testapi.TestAPI, roleApi testapi.TestAPI, teamAPI testapi.TestAPI, configAPI testapi.ConfigAPI) *config.FlantFlowConfig {
	cfg := BaseConfigureFlantFlow(tenantAPI, roleApi, configAPI)

	teamL1 := CreateRandomTeam(teamAPI)
	teamMk8s := CreateRandomTeam(teamAPI)
	teamOkmeter := CreateRandomTeam(teamAPI)
	teams := map[string]string{
		config.L1:      teamL1.UUID,
		config.Mk8s:    teamMk8s.UUID,
		config.Okmeter: teamOkmeter.UUID,
	}
	configAPI.ConfigureExtensionFlantFlowSpecificTeams(teams)
	cfg.SpecificTeams = teams
	return cfg
}

func BaseConfigureFlantFlow(tenantAPI testapi.TestAPI, roleAPI testapi.TestAPI, configAPI testapi.ConfigAPI) *config.FlantFlowConfig {
	tenant := iam_specs.CreateRandomTenant(tenantAPI)
	configAPI.ConfigureExtensionFlantFlowFlantTenantUUID(tenant.UUID)
	r1 := iam_specs.CreateRandomRole(roleAPI)
	configAPI.ConfigureExtensionFlantFlowSpecificRoles(map[string]string{"todo_in_future": r1.Name}) // TODO fil later

	return &config.FlantFlowConfig{
		FlantTenantUUID: tenant.UUID,
		SpecificTeams:   map[string]string{},
		SpecificRoles:   map[string]string{},
	}
}
