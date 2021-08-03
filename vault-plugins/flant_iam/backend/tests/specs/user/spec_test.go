package user

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_userCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewUserAPI(&backend)
	TenantAPI = api.NewTenantAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: User")
}
