package rolebindingapproval

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test(t *testing.T) {
	backend := api.TestBackend()
	TestTenantAPI = api.NewTenantAPI(&backend)
	TestRoleBindingAPI = api.NewRoleBindingAPI(&backend)
	TestRoleBindingApprovalAPI = api.NewRoleBindingApprovalAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Role binding approval")
}
