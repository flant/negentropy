package specs

import (
	"encoding/json"
	"net/url"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func CreateFlantTenant(tenantAPI testapi.TestAPI) iam_model.Tenant {
	createPayload := map[string]interface{}{
		"identifier":    "Identifier_FLANT",
		"version":       "v1",
		"feature_flags": nil,
		"uuid":          fixtures.FlantUUID,
	}
	var createdData gjson.Result
	tenantAPI.CreatePrivileged(testapi.Params{
		"expectPayload": func(json gjson.Result) {
			createdData = json
		},
	}, nil, createPayload)
	rawTenant := createdData.Get("tenant")
	data := []byte(rawTenant.String())
	var tenant iam_model.Tenant
	err := json.Unmarshal(data, &tenant)
	Expect(err).ToNot(HaveOccurred())
	return tenant
}

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
	createPayload["tenant_uuid"] = clientID
	params := testapi.Params{
		"client": clientID,
	}
	createData := projectAPI.Create(params, url.Values{}, createPayload)
	rawProject := createData.Get("project")
	data := []byte(rawProject.String())
	var project model.Project
	err := json.Unmarshal(data, &project)
	Expect(err).ToNot(HaveOccurred())
	return project
}

func CreateRandomContact(contactAPI testapi.TestAPI, clientID model.TeamUUID) model.Contact {
	createPayload := fixtures.RandomContactCreatePayload()
	createPayload["tenant_uuid"] = clientID
	createdData := contactAPI.Create(testapi.Params{
		"client": clientID,
	}, nil, createPayload)
	rawContact := createdData.Get("contact")
	data := []byte(rawContact.String())
	var contact model.Contact
	err := json.Unmarshal(data, &contact)
	Expect(err).ToNot(HaveOccurred())
	return contact
}
