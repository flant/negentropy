package role

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/role"
)

func Test_roleCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	role.TestAPI = lib.NewRoleAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Role")
}
