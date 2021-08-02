package role

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/tenant"
)

func Test_tenantCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	tenant.TestAPI = lib.NewTenantAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Tenant")
}
