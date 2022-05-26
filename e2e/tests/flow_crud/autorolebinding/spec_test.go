package autorb

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	autorb "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs/autorolebinding"
)

func Test_teammateCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	autorb.RoleAPI = lib.NewRoleAPI(rootClient)
	autorb.TeamAPI = lib.NewFlowTeamAPI(rootClient)
	autorb.TenantAPI = lib.NewTenantAPI(rootClient)
	autorb.ConfigAPI = lib.NewHttpClientBasedConfigAPI(rootClient)

	autorb.GroupAPI = lib.NewGroupAPI(rootClient)
	autorb.RoleBindingAPI = lib.NewRoleBindingAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "flant_flow: autorolebindings")
}
