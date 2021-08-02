package rolebindingapproval

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/rolebindingapproval"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	rolebindingapproval.TestTenantAPI = lib.NewTenantAPI(rootClient)
	rolebindingapproval.TestRoleBindingAPI = lib.NewRoleBindingAPI(rootClient)
	rolebindingapproval.TestRoleBindingApprovalAPI = lib.NewRoleBindingApprovalAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Role binding approval")
}
