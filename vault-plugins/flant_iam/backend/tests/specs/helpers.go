package specs

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func IsMapSubsetOfSetExceptKeys(mapa map[string]interface{}, set gjson.Result, keys ...string) {
	bytes, err := json.Marshal(mapa)
	Expect(err).ToNot(HaveOccurred())
	subset := gjson.Parse(string(bytes))
	IsSubsetExceptKeys(subset, set, keys...)
}

func IsSubsetExceptKeys(subset gjson.Result, set gjson.Result, keys ...string) {
	setMap := set.Map()
	subsetMap := subset.Map()
	for _, key := range keys {
		subsetMap[key] = setMap[key]
	}
	for k, v := range subsetMap {
		Expect(v.String()).To(Equal(setMap[k].String()), "field:", k)
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

func CheckObjectArrayForUUID(array []gjson.Result, uuid string, shouldContains bool) {
	found := false
	for i := range array {
		Expect(array[i].Map()).To(HaveKey("uuid"))
		if array[i].Map()["uuid"].String() == uuid {
			found = true
			break
		}
	}
	if shouldContains {
		Expect(found).To(BeTrue())
	} else {
		Expect(found).To(BeFalse())
	}
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
	createPayload["identifier"] = "user_" + uuid.New()
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

func CreateRandomEmptyGroup(groupAPI api.TestAPI, tenantID model.TenantUUID) model.Group {
	createPayload := fixtures.RandomGroupCreatePayload()
	createPayload["tenant_uuid"] = tenantID
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

func membersToSliceOfMaps(members model.Members) []map[string]interface{} {
	result := []map[string]interface{}{}
	for _, userUUID := range members.Users {
		result = append(result, map[string]interface{}{
			"type": "user",
			"uuid": userUUID,
		})
	}
	for _, saUUID := range members.ServiceAccounts {
		result = append(result, map[string]interface{}{
			"type": "service_account",
			"uuid": saUUID,
		})
	}
	for _, gUUID := range members.Groups {
		result = append(result, map[string]interface{}{
			"type": "group",
			"uuid": gUUID,
		})
	}
	return result
}

func CreateRandomGroupWithMembers(groupAPI api.TestAPI, tenantID model.TenantUUID, members model.Members) model.Group {
	createPayload := fixtures.RandomGroupCreatePayload()
	createPayload["tenant_uuid"] = tenantID
	createPayload["members"] = membersToSliceOfMaps(members)
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
		[]string{"ssh.open"})
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
	rolesData := roleAPI.List(api.Params{}, nil).Get("roles")
	roleNames := map[string]struct{}{}
	for _, roleData := range rolesData.Array() {
		roleNames[roleData.Get("name").String()] = struct{}{}
	}
	for _, r := range roles {
		if _, found := roleNames[r.Name]; !found {
			createPayload := fixtures.RoleCreatePayload(r)
			params := api.Params{}
			roleAPI.Create(params, url.Values{}, createPayload)
		}
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
	delete(createPayload, "valid_till")
	createPayload["ttl"] = rb.ValidTill - time.Now().Unix()
	createData := rolebindingAPI.Create(params, url.Values{}, createPayload)
	rawRoleBinding := createData.Get("role_binding")
	data := []byte(rawRoleBinding.String())
	var rbd usecase.DenormalizedRoleBinding
	err := json.Unmarshal(data, &rbd)
	Expect(err).ToNot(HaveOccurred())

	return model.RoleBinding{
		ArchiveMark:     rbd.ArchiveMark,
		UUID:            rbd.UUID,
		TenantUUID:      rbd.TenantUUID,
		Version:         rbd.Version,
		Description:     rbd.Description,
		ValidTill:       rbd.ValidTill,
		RequireMFA:      rbd.RequireMFA,
		Users:           nil,
		Groups:          nil,
		ServiceAccounts: nil,
		Members:         MapMembers(rbd.Members),
		AnyProject:      rbd.AnyProject,
		Projects:        MapProjects(rbd.Projects),
		Roles:           rbd.Roles,
		Origin:          rbd.Origin,
		Extensions:      nil,
	}
}

func MapMembers(members []usecase.DenormalizedMemberNotation) []model.MemberNotation {
	var result []model.MemberNotation
	for _, m := range members {
		result = append(result, model.MemberNotation{
			Type: m.Type,
			UUID: m.UUID,
		})
	}
	return result
}

func MapProjects(projects []usecase.ProjectUUIDWithIdentifier) []model.ProjectUUID {
	var result []model.ProjectUUID
	for _, m := range projects {
		result = append(result, m.UUID)
	}
	return result
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

func ShareGroupToTenant(identitySharingAPI api.TestAPI, group model.Group, targetTenantUUID model.TenantUUID) gjson.Result {
	params := api.Params{
		"tenant": group.TenantUUID,
	}
	data := map[string]interface{}{
		"destination_tenant_uuid": targetTenantUUID,
		"groups":                  []string{group.UUID},
	}
	is := identitySharingAPI.Create(params, url.Values{}, data).Get("identity_sharing")
	return is
}

func CreateRandomServer(serverApi api.TestAPI, tenantUUID string, projectUUID string) (srv ext_model.Server, saJWT string) {
	payload := api.Params{
		"identifier": "server_" + uuid.New(),
		"labels":     map[string]string{"l1": "v1"},
	}

	params := api.Params{
		"tenant":  tenantUUID,
		"project": projectUUID,
	}
	createdData := serverApi.Create(params, nil, payload)
	srvUUID := createdData.Get("uuid").String()
	saJWT = createdData.Get("multipassJWT").String()
	readData := serverApi.Read(api.Params{
		"tenant":       tenantUUID,
		"project":      projectUUID,
		"server":       srvUUID,
		"expectStatus": api.ExpectExactStatus(http.StatusOK),
	}, nil)
	rawSrv := readData.Get("server")
	data := []byte(rawSrv.String())
	var server ext_model.Server
	err := json.Unmarshal(data, &server)
	Expect(err).ToNot(HaveOccurred())
	return server, saJWT
}
