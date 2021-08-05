package tenant_featureflag

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_tenantFeatureFlagCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestTenantAPI = api.NewTenantAPI(&backend)
	TestFeatureFlagAPI = api.NewFeatureFlagAPI(&backend)
	TestTenantFFApi = api.NewTenantFeatureFlagAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Tenant Feature Flags")
}
