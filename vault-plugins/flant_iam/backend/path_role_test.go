package backend

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tidwall/gjson"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/sdk/physical/inmem"
	"github.com/stretchr/testify/assert"
)

func Test_RolePathCreate(t *testing.T) {
	b := testRoleBackend(t)
	roleName := "ssh"

	resp := createRole(t, roleName, b)

	role := roleMap(t, resp)
	assert.Contains(t, role, "name")
	assert.Equal(t, role["name"], roleName)
	assert.Contains(t, role, "archiving_timestamp")
	assert.Equal(t, role["archiving_timestamp"], 0.0)
}

func Test_RoleDelete(t *testing.T) {
	b := testRoleBackend(t)
	roleName := "ssh"
	resp := createRole(t, roleName, b)

	resp = deleteRole(t, roleName, b)

	assert.Contains(t, resp.Data, "http_status_code")
	status, ok := (resp.Data["http_status_code"]).(int)
	assert.True(t, ok)
	assert.Equal(t, status, 204)
	resp = readRole(t, roleName, b)
	role := roleMap(t, resp)
	assert.Contains(t, role, "name")
	assert.Equal(t, role["name"], roleName)
	assert.Contains(t, role, "archiving_timestamp")
	assert.Greater(t, role["archiving_timestamp"], 0.0)
}

func Test_RoleActiveList(t *testing.T) {
	b := testRoleBackend(t)
	roleName := "ssh"
	resp := createRole(t, roleName, b)

	resp = listRoles(t, false, b)

	assert.Contains(t, resp.Data, "http_raw_body")
	data := gjson.Parse(resp.Data["http_raw_body"].(string))
	assert.Contains(t, data.Map(), "data")
	assert.Contains(t, data.Get("data").Map(), "names")
	assert.Len(t, data.Get("data.names").Array(), 1)
}

func Test_RoleArchivedList(t *testing.T) {
	b := testRoleBackend(t)
	roleName := "ssh"
	resp := createRole(t, roleName, b)
	resp = listRoles(t, false, b)
	assert.Contains(t, resp.Data, "http_raw_body")
	data := gjson.Parse(resp.Data["http_raw_body"].(string))
	assert.Contains(t, data.Map(), "data")
	assert.Contains(t, data.Get("data").Map(), "names")
	assert.Len(t, data.Get("data.names").Array(), 1)
	resp = deleteRole(t, roleName, b)
	assert.Contains(t, resp.Data, "http_status_code")
	status, ok := (resp.Data["http_status_code"]).(int)
	assert.True(t, ok)
	assert.Equal(t, status, 204)
	resp = listRoles(t, false, b)
	assert.Contains(t, resp.Data, "http_raw_body")
	data = gjson.Parse(resp.Data["http_raw_body"].(string))
	assert.Contains(t, data.Map(), "data")
	assert.Contains(t, data.Get("data").Map(), "names")
	assert.Len(t, data.Get("data.names").Array(), 0)

	resp = listRoles(t, true, b)

	assert.Contains(t, resp.Data, "http_raw_body")
	data = gjson.Parse(resp.Data["http_raw_body"].(string))
	assert.Contains(t, data.Map(), "data")
	assert.Contains(t, data.Get("data").Map(), "names")
	assert.Len(t, data.Get("data.names").Array(), 1)
}

func listRoles(t *testing.T, showArchived bool, b logical.Backend) *logical.Response {
	data := map[string]interface{}{}
	if showArchived {
		data["show_archived"] = true
	}
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "role/",
		Data:      data,
	})
	assert.NoError(t, err)

	return resp
}

func roleMap(t *testing.T, resp *logical.Response) map[string]interface{} {
	assert.Contains(t, resp.Data, "http_raw_body")
	rawBody, ok := (resp.Data["http_raw_body"]).(string)
	assert.True(t, ok)
	var responseMap map[string]interface{}
	err := json.Unmarshal([]byte(rawBody), &responseMap)
	assert.NoError(t, err)
	assert.Contains(t, responseMap, "data")
	dataMap, ok := responseMap["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, dataMap, "role")
	role, ok := (dataMap["role"]).(map[string]interface{})
	assert.True(t, ok)
	return role
}

func createRole(t *testing.T, roleName string, b logical.Backend) *logical.Response {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "role",
		Data: map[string]interface{}{
			"name":                         roleName,
			"description":                  "test_role",
			"scope":                        "project",
			"options_schema":               "<1>",
			"require_one_of_feature_flags": []string{},
			"archived":                     true,
		},
	})
	assert.NoError(t, err)
	return resp
}

func deleteRole(t *testing.T, roleName string, b logical.Backend) *logical.Response {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "role/" + roleName,
		Data:      map[string]interface{}{},
	})
	assert.NoError(t, err)
	return resp
}

func readRole(t *testing.T, roleName string, b logical.Backend) *logical.Response {
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "role/" + roleName,
		Data:      map[string]interface{}{},
	})
	assert.NoError(t, err)
	return resp
}

func testRoleBackend(t *testing.T) logical.Backend {
	config := logical.TestBackendConfig()
	testPhisicalBackend, err := inmem.NewInmemHA(map[string]string{}, config.Logger)
	assert.NoError(t, err)
	config.StorageView = logical.NewStorageView(logical.NewLogicalStorage(testPhisicalBackend), "")
	b, err := Factory(context.Background(), config)
	assert.NoError(t, err)
	return b
}
