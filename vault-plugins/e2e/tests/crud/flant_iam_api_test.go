package tenant

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	apispecs "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
)

func Test_RoleCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	apispecs.RoleCrudSpec(lib.NewRoleAPI(rootClient))
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Role")
}

func Test_FeatureFlagCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	apispecs.FeatureFlagCrudSpec(lib.NewFeatureFlagAPI(rootClient))
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: FeatureFlag")
}

func Test_TenantCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	apispecs.TenantCrudSpec(lib.NewTenantAPI(rootClient))
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Tenant")
}
