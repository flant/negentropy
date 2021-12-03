package contact

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/api"
)

func Test_contactCRUD(t *testing.T) {
	backend, storage := testapi.TestBackendWithStorage()
	TestAPI = api.NewContactAPI(&backend, &storage)
	ClientAPI = api.NewClientAPI(&backend, &storage)
	ProjectAPI = api.NewProjectAPI(&backend, &storage)
	TenantAPI = testapi.NewTenantAPI(&backend)
	RoleAPI = testapi.NewRoleAPI(&backend)
	TeamAPI = api.NewTeamAPI(&backend, &storage)
	ConfigAPI = testapi.NewBackendBasedConfigAPI(&backend, &storage)

	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Contact")
}
