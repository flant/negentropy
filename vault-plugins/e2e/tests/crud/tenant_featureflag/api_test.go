package tenant_featureflag

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/tenant_featureflag"
)

func Test_tenantFeatureFlagCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	tenant_featureflag.TestTenantAPI = lib.NewTenantAPI(rootClient)
	tenant_featureflag.TestFeatureFlagAPI = lib.NewFeatureFlagAPI(rootClient)
	tenant_featureflag.TestTenantFFApi = lib.NewTenantFeatureFlagAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Tenant Feature Flags")
}
