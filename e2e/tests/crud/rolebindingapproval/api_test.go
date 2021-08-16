package rolebindingapproval

import (
	"testing"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/rolebindingapproval"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_rolebindingApprovalCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	rolebindingapproval.TenantAPI = lib.NewTenantAPI(rootClient)
	rolebindingapproval.RoleBindingAPI = lib.NewRoleBindingAPI(rootClient)
	rolebindingapproval.TestAPI = lib.NewRoleBindingApprovalAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Role binding approval")
}
