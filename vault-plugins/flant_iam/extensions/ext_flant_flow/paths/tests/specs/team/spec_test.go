package team

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/api"
)

func Test_teamCRUD(t *testing.T) {
	backend, storage := testapi.TestBackendWithStorage()
	TestAPI = api.NewTeamAPI(&backend, &storage)
	TenantAPI = testapi.NewTenantAPI(&backend)
	RoleAPI = testapi.NewRoleAPI(&backend)
	ConfigAPI = testapi.NewBackendBasedConfigAPI(&backend, &storage)

	GroupAPI = testapi.NewGroupAPI(&backend)
	TeammateAPI = api.NewTeammateAPI(&backend, &storage)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Team")
}
