package user

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/user"
)

func Test_userCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	user.TenantAPI = lib.NewTenantAPI(rootClient)
	user.TestAPI = lib.NewUserAPI(rootClient)
	user.IdentitySharingAPI = lib.NewIdentitySharingAPI(rootClient)
	user.GroupAPI = lib.NewGroupAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: User")
}
