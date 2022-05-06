package identitysharing

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/identitysharing"
)

func Test_identitySharingCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	identitysharing.TenantAPI = lib.NewTenantAPI(rootClient)
	identitysharing.GroupAPI = lib.NewGroupAPI(rootClient)
	identitysharing.TestAPI = lib.NewIdentitySharingAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: IdentitySharing")
}
