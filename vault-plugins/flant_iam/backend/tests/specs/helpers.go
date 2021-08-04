package specs

import (
	"encoding/json"
	"net/url"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
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

func CreateRandomTenant(tenantsAPI api.TestAPI) model.Tenant {
	createPayload := fixtures.RandomTenantCreatePayload()

	var createdData gjson.Result
	tenantsAPI.Create(api.Params{
		"expectPayload": func(json gjson.Result) {
			createdData = json
		},
	}, nil, createPayload)
	rawTenant := createdData.Get("tenant")
	data := []byte(rawTenant.String())
	var tenant model.Tenant
	err := json.Unmarshal(data, &tenant) //nolint:errcheck
	Expect(err).ToNot(HaveOccurred())
	return tenant
}

func CreateRandomUser(userAPI api.TestAPI, tenantID model.TenantUUID) model.User {
	createPayload := fixtures.RandomUserCreatePayload()
	createPayload["tenant_uuid"] = tenantID
	createdData := userAPI.Create(api.Params{
		"tenant": tenantID,
	}, nil, createPayload)
	rawUser := createdData.Get("user")
	data := []byte(rawUser.String())
	var user model.User
	err := json.Unmarshal(data, &user) //nolint:errcheck
	Expect(err).ToNot(HaveOccurred())
	return user
}

func CreateRandomGroup(groupAPI api.TestAPI, tenantID model.TenantUUID, userID model.UserUUID) model.Group {
	createPayload := fixtures.RandomGroupCreatePayload()
	createPayload["tenant_uuid"] = tenantID
	createPayload["members"] = map[string]interface{}{
		"type": "user",
		"uuid": userID,
	}

	params := api.Params{
		"tenant": tenantID,
	}
	createdData := groupAPI.Create(params, url.Values{}, createPayload)
	rawGroup := createdData.Get("group")
	data := []byte(rawGroup.String())
	var group model.Group
	err := json.Unmarshal(data, &group) //nolint:errcheck
	Expect(err).ToNot(HaveOccurred())
	return group
}

func CreateRandomUserMultipass(userMultipassAPI api.TestAPI, tenantID model.TenantUUID, userID model.UserUUID) model.Multipass {
	createPayload := fixtures.RandomUserMultipassCreatePayload()
	createPayload["tenant_uuid"] = tenantID
	createPayload["owner_uuid"] = userID

	params := api.Params{
		"tenant": tenantID,
		"user":   userID,
	}
	createData := userMultipassAPI.Create(params, url.Values{}, createPayload)
	rawMultipass := createData.Get("multipass")
	data := []byte(rawMultipass.String())
	var multipass model.Multipass
	err := json.Unmarshal(data, &multipass) //nolint:errcheck
	Expect(err).ToNot(HaveOccurred())
	return multipass
}

func CreateRandomRole(roleAPI api.TestAPI) model.Role {
	createPayload := fixtures.RandomRoleCreatePayload()

	params := api.Params{}
	createData := roleAPI.Create(params, url.Values{}, createPayload)
	rawRole := createData.Get("role")
	data := []byte(rawRole.String())
	var role model.Role
	err := json.Unmarshal(data, &role) //nolint:errcheck
	Expect(err).ToNot(HaveOccurred())
	return role
}

func CreateRandomProject(ProjectAPI api.TestAPI, tenantID model.TenantUUID) model.Project {
	createPayload := fixtures.RandomGroupCreatePayload()
	createPayload["tenant_uuid"] = tenantID

	params := api.Params{
		"tenant": tenantID,
	}
	createData := ProjectAPI.Create(params, url.Values{}, createPayload)

	rawProject := createData.Get("project")
	data := []byte(rawProject.String())
	var project model.Project
	err := json.Unmarshal(data, &project) //nolint:errcheck
	Expect(err).ToNot(HaveOccurred())
	return project
}
