package tenant_featureflag

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/tenant_featureflag"
)

func Test_tenantFeatureFlagCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	tenant_featureflag.TenantAPI = lib.NewTenantAPI(rootClient)
	tenant_featureflag.FeatureFlagAPI = lib.NewFeatureFlagAPI(rootClient)
	tenant_featureflag.TestApi = lib.NewTenantFeatureFlagAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Tenant Feature Flags")
}
