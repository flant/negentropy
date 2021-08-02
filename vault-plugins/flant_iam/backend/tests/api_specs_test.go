package tests

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	backend := api.TestBackend()
	specs.RoleAPI = api.NewRoleAPI(&backend)
	RunSpecs(t, "CRUD: Role")
}
