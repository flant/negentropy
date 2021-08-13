package specs

import (
	"encoding/json"
	"net/url"
	"time"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
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
	err := json.Unmarshal(data, &tenant)
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
	err := json.Unmarshal(data, &user)
	Expect(err).ToNot(HaveOccurred())
	return user
}

func CreateRandomGroupWithUser(groupAPI api.TestAPI, tenantID model.TenantUUID, userID model.UserUUID) model.Group {
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
	err := json.Unmarshal(data, &group)
	Expect(err).ToNot(HaveOccurred())
	return group
}

func CreateRandomUserMultipass(userMultipassAPI api.TestAPI, user model.User) model.Multipass {
	multipass, _ := CreateUserMultipass(userMultipassAPI, user,
		"desc - "+uuid.New(),
		100*time.Second,
		1000*time.Second,
		[]string{"ssh"})
	return multipass
}

// return Mutipass model and JWT
func CreateUserMultipass(userMultipassAPI api.TestAPI, user model.User, description string,
	ttl time.Duration, maxTTL time.Duration, roles []model.RoleName) (model.Multipass, string) {
	createPayload := map[string]interface{}{
		"tenant_uuid":   user.TenantUUID,
		"owner_uuid":    user.UUID,
		"owner_type":    model.UserType,
		"description":   description,
		"ttl":           ttl,
		"max_ttl":       maxTTL,
		"allowed_roles": roles,
	}
	params := api.Params{
		"tenant": user.TenantUUID,
		"user":   user.UUID,
	}
	createData := userMultipassAPI.Create(params, url.Values{}, createPayload)
	rawMultipass := createData.Get("multipass")
	data := []byte(rawMultipass.String())
	var multipass model.Multipass
	err := json.Unmarshal(data, &multipass)
	Expect(err).ToNot(HaveOccurred())
	return multipass, createData.Get("token").String()
}

func CreateRandomRole(roleAPI api.TestAPI) model.Role {
	createPayload := fixtures.RandomRoleCreatePayload()
	params := api.Params{}
	createData := roleAPI.Create(params, url.Values{}, createPayload)
	rawRole := createData.Get("role")
	data := []byte(rawRole.String())
	var role model.Role
	err := json.Unmarshal(data, &role)
	Expect(err).ToNot(HaveOccurred())
	return role
}

func CreateRandomProject(projectAPI api.TestAPI, tenantID model.TenantUUID) model.Project {
	createPayload := fixtures.RandomGroupCreatePayload()
	createPayload["tenant_uuid"] = tenantID
	params := api.Params{
		"tenant": tenantID,
	}
	createData := projectAPI.Create(params, url.Values{}, createPayload)
	rawProject := createData.Get("project")
	data := []byte(rawProject.String())
	var project model.Project
	err := json.Unmarshal(data, &project)
	Expect(err).ToNot(HaveOccurred())
	return project
}

func CreateRoleBinding(rolebindingAPI api.TestAPI, rb model.RoleBinding) model.RoleBinding {
	params := api.Params{
		"tenant": rb.TenantUUID,
	}
	bytes, _ := json.Marshal(rb)
	var createPayload map[string]interface{}
	json.Unmarshal(bytes, &createPayload) //nolint:errcheck
	createData := rolebindingAPI.Create(params, url.Values{}, createPayload)
	rawRoleBinding := createData.Get("role_binding")
	data := []byte(rawRoleBinding.String())
	var roleBinding model.RoleBinding
	err := json.Unmarshal(data, &roleBinding)
	Expect(err).ToNot(HaveOccurred())
	return roleBinding
}

type ServerRegistrationResult struct {
	MultipassJWT string `json:"multipassJWT"`
	ServerUUID   string `json:"uuid"`
}

func RegisterServer(serverAPI api.TestAPI, server model2.Server) ServerRegistrationResult {
	params := api.Params{
		"tenant":       server.TenantUUID,
		"project":      server.ProjectUUID,
		"expectStatus": api.ExpectExactStatus(200),
	}
	bytes, _ := json.Marshal(server)
	var createPayload map[string]interface{}
	json.Unmarshal(bytes, &createPayload) //nolint:errcheck
	createData := serverAPI.Create(params, url.Values{}, createPayload)
	data := []byte(createData.String())
	var createdServer ServerRegistrationResult
	err := json.Unmarshal(data, &createdServer)
	Expect(err).ToNot(HaveOccurred())
	return createdServer
}

func UpdateConnectionInfo(connectionInfoAPI api.TestAPI, server model2.Server, info model2.ConnectionInfo) model2.Server {
	params := api.Params{
		"tenant":       server.TenantUUID,
		"project":      server.ProjectUUID,
		"server":       server.UUID,
		"expectStatus": api.ExpectExactStatus(200),
	}
	bytes, _ := json.Marshal(info)
	var createPayload map[string]interface{}
	json.Unmarshal(bytes, &createPayload) //nolint:errcheck
	createData := connectionInfoAPI.Update(params, url.Values{}, createPayload)
	data := []byte(createData.Get("server").String())
	var resultServer model2.Server
	err := json.Unmarshal(data, &resultServer)
	Expect(err).ToNot(HaveOccurred())
	return resultServer
}

// return Mutipass model and JWT
func CreateServiceAccountMultipass(serviceAccountMultipassAPI api.TestAPI, serviceAccount model.ServiceAccount, description string,
	ttl time.Duration, maxTTL time.Duration, roles []model.RoleName) (model.Multipass, string) {
	createPayload := map[string]interface{}{
		"tenant_uuid":   serviceAccount.TenantUUID,
		"owner_uuid":    serviceAccount.UUID,
		"owner_type":    model.ServiceAccountType,
		"description":   description,
		"ttl":           ttl,
		"max_ttl":       maxTTL,
		"allowed_roles": roles,
	}
	params := api.Params{
		"tenant": serviceAccount.TenantUUID,
		"user":   serviceAccount.UUID,
	}
	createData := serviceAccountMultipassAPI.Create(params, url.Values{}, createPayload)
	rawMultipass := createData.Get("multipass")
	data := []byte(rawMultipass.String())
	var multipass model.Multipass
	err := json.Unmarshal(data, &multipass)
	Expect(err).ToNot(HaveOccurred())
	return multipass, createData.Get("token").String()
}
