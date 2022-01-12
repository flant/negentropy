package rolebinding

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/rolebinding"
)

func Test_rolebindingApprovalCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	rolebinding.TenantAPI = lib.NewTenantAPI(rootClient)
	rolebinding.UserAPI = lib.NewUserAPI(rootClient)
	rolebinding.ServiceAccountAPI = lib.NewServiceAccountAPI(rootClient)
	rolebinding.GroupAPI = lib.NewGroupAPI(rootClient)
	rolebinding.RoleAPI = lib.NewRoleAPI(rootClient)
	rolebinding.TestAPI = lib.NewRoleBindingAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Role binding")
}
