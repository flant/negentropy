package rolebindingapproval

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_rolebindingApprovalCRUD(t *testing.T) {
	backend := api.TestBackend()
	TenantAPI = api.NewTenantAPI(&backend)
	UserAPI = api.NewUserAPI(&backend)
	ServiceAccountAPI = api.NewServiceAccountAPI(&backend)
	GroupAPI = api.NewGroupAPI(&backend)
	RoleAPI = api.NewRoleAPI(&backend)
	RoleBindingAPI = api.NewRoleBindingAPI(&backend)
	TestAPI = api.NewRoleBindingApprovalAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_iam: Role binding approval")
}
