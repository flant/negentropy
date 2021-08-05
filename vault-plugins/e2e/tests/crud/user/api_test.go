package user

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/user"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
)

func Test_userCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	user.TenantAPI = lib.NewTenantAPI(rootClient)
	user.TestAPI = lib.NewUserAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: User")
}
