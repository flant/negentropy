package tenant

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apispecs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
)

func Test(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	apispecs.RoleAPI = lib.NewRoleAPI(rootClient)

	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Role")
}
