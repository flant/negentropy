package role

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_roleCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewRoleAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Role")
}
