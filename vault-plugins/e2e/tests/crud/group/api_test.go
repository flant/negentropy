package group

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/group"
)

func Test_tenantCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	group.TenantAPI = lib.NewTenantAPI(rootClient)
	group.UserAPI = lib.NewUserAPI(rootClient)
	group.TestAPI = lib.NewGroupAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Group")
}
