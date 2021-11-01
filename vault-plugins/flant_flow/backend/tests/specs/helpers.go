package specs

import (
	"encoding/json"
	"net/url"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_flow/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
)

func IsSubsetExceptKeys(subset gjson.Result, set gjson.Result, keys ...string) {
	setMap := set.Map()
	subsetMap := subset.Map()
	for _, key := range keys {
		subsetMap[key] = setMap[key]
	}
	for k, v := range subsetMap {
		Expect(v).To(Equal(setMap[k]))
	}
}

func CheckArrayContainsElement(array []gjson.Result, element gjson.Result) {
	var mapArray []map[string]gjson.Result
	for i := range array {
		mapArray = append(mapArray, array[i].Map())
	}
	Expect(mapArray).To(ContainElement(element.Map()))
}

func CheckArrayContainsElementByUUIDExceptKeys(array []gjson.Result, element gjson.Result, keys ...string) {
	Expect(element.Map()).To(HaveKey("uuid"))
	found := false
	for i := range array {
		Expect(array[i].Map()).To(HaveKey("uuid"))
		if array[i].Map()["uuid"] == element.Map()["uuid"] {
			IsSubsetExceptKeys(element, array[i], keys...)
			found = true
			break
		}
	}
	Expect(found).To(BeTrue())
}

func ConvertToGJSON(object interface{}) gjson.Result {
	bytes, err := json.Marshal(object)
	Expect(err).ToNot(HaveOccurred())
	return gjson.Parse(string(bytes))
}

func CreateRandomClient(clientsAPI api.TestAPI) model.Client {
	createPayload := fixtures.RandomClientCreatePayload()
	var createdData gjson.Result
	clientsAPI.Create(api.Params{
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

func CreateRandomTeam(teamAPI api.TestAPI) model.Team {
	createPayload := fixtures.RandomTeamCreatePayload()
	var createdData gjson.Result
	teamAPI.Create(api.Params{
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

func CreateRandomTeammate(teamateAPI api.TestAPI, teamtID model.TeamUUID) model.Teammate {
	createPayload := fixtures.RandomTeammateCreatePayload()
	createPayload["team_uuid"] = teamtID
	createdData := teamateAPI.Create(api.Params{
		"team": teamtID,
	}, nil, createPayload)
	rawTemmate := createdData.Get("teammate")
	data := []byte(rawTemmate.String())
	var teammate model.Teammate
	err := json.Unmarshal(data, &teammate)
	Expect(err).ToNot(HaveOccurred())
	return teammate
}

func CreateRandomProject(projectAPI api.TestAPI, clientID model.ClientUUID) model.Project {
	createPayload := fixtures.RandomProjectCreatePayload()
	createPayload["tenant_uuid"] = clientID
	params := api.Params{
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

func CreateRandomContact(contactAPI api.TestAPI, clientID model.TeamUUID) model.Contact {
	createPayload := fixtures.RandomContactCreatePayload()
	createPayload["tenant_uuid"] = clientID
	createdData := contactAPI.Create(api.Params{
		"client": clientID,
	}, nil, createPayload)
	rawContact := createdData.Get("contact")
	data := []byte(rawContact.String())
	var contact model.Contact
	err := json.Unmarshal(data, &contact)
	Expect(err).ToNot(HaveOccurred())
	return contact
}
