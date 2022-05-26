package autorb

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/api"
)

func Test_teammateCRUD(t *testing.T) {
	backend, storage := testapi.TestBackendWithStorage()
	RoleAPI = testapi.NewRoleAPI(&backend)
	TeamAPI = api.NewTeamAPI(&backend, &storage)
	TenantAPI = testapi.NewTenantAPI(&backend)
	ConfigAPI = testapi.NewBackendBasedConfigAPI(&backend, &storage)

	GroupAPI = testapi.NewGroupAPI(&backend)
	RoleBindingAPI = testapi.NewRoleBindingAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "flow: autorolebindings")
}
