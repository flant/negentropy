package rolebinding

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_rolebindingCRUD(t *testing.T) {
	backend := api.TestBackend()
	TenantAPI = api.NewTenantAPI(&backend)
	UserAPI = api.NewUserAPI(&backend)
	ServiceAccountAPI = api.NewServiceAccountAPI(&backend)
	GroupAPI = api.NewGroupAPI(&backend)
	RoleAPI = api.NewRoleAPI(&backend)
	TestAPI = api.NewRoleBindingAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_iam: Role binding")
}
