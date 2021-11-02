package tenant_featureflag

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_tenantFeatureFlagCRUD(t *testing.T) {
	backend := api.TestBackend()
	TenantAPI = api.NewTenantAPI(&backend)
	FeatureFlagAPI = api.NewFeatureFlagAPI(&backend)
	TestAPI = api.NewTenantFeatureFlagAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_iam: Tenant Feature Flags")
}
