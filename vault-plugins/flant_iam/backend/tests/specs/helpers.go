package specs

import (
	"encoding/json"
	"net/url"
	"time"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func IsSubsetExceptKeys(subset gjson.Result, set gjson.Result, keys ...string) {
	setMap := set.Map()
	subsetMap := subset.Map()
	for _, key := range keys {
		subsetMap[key] = setMap[key]
	}
	for k, v := range subsetMap {
		Expect(v.String()).To(Equal(setMap[k].String()))
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
		if array[i].Map()["uuid"].String() == element.Map()["uuid"].String() {
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

func CreateRoles(roleAPI api.TestAPI, roles ...model.Role) {
	for _, r := range roles {
		createPayload := fixtures.RoleCreatePayload(r)
		params := api.Params{}
		// don't check errors, as global roles can be already created
		roleAPI.Create(params, url.Values{}, createPayload)
	}
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

// CreateServiceAccountMultipass returns Mutipass model and JWT
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

func CreateRandomServiceAccount(serviceAccountAPI api.TestAPI, tenantUUID model.TenantUUID) model.ServiceAccount {
	createPayload := fixtures.RandomServiceAccountCreatePayload()
	createPayload["tenant_uuid"] = tenantUUID
	params := api.Params{
		"tenant": tenantUUID,
	}
	createData := serviceAccountAPI.Create(params, url.Values{}, createPayload)
	rawServiceAccount := createData.Get("service_account")
	data := []byte(rawServiceAccount.String())
	var serviceAccount model.ServiceAccount
	err := json.Unmarshal(data, &serviceAccount)
	Expect(err).ToNot(HaveOccurred())
	return serviceAccount
}

// CreateServiceAccountPassword returns ServiceAccountPassword model
func CreateServiceAccountPassword(serviceAccountPasswordAPI api.TestAPI, serviceAccount model.ServiceAccount, description string,
	ttl time.Duration, roles []model.RoleName) model.ServiceAccountPassword {
	createPayload := map[string]interface{}{
		"tenant_uuid": serviceAccount.TenantUUID,
		"owner_uuid":  serviceAccount.UUID,
		"description": description,
		// "allowed_cidrs":"",
		"allowed_roles": roles,
		"ttl":           ttl,
	}
	params := api.Params{
		"tenant":          serviceAccount.TenantUUID,
		"service_account": serviceAccount.UUID,
	}
	createData := serviceAccountPasswordAPI.Create(params, url.Values{}, createPayload)
	rawPass := createData.Get("password")
	data := []byte(rawPass.String())
	var password model.ServiceAccountPassword
	err := json.Unmarshal(data, &password)
	Expect(err).ToNot(HaveOccurred())
	return password
}
