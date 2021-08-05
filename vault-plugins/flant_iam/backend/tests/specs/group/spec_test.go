package group

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_groupCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewGroupAPI(&backend)
	TenantAPI = api.NewTenantAPI(&backend)
	UserAPI = api.NewUserAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Group")
}
