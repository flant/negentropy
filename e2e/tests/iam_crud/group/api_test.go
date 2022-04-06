package group

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/group"
)

func Test_groupCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	group.TenantAPI = lib.NewTenantAPI(rootClient)
	group.UserAPI = lib.NewUserAPI(rootClient)
	group.TestAPI = lib.NewGroupAPI(rootClient)
	group.IdentitySharingAPI = lib.NewIdentitySharingAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Group")
}
