package tests

import (
	"testing"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_roleCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	backend := api.TestBackend()
	specs.RoleCrudSpec(api.NewRoleAPI(&backend))
	RunSpecs(t, "CRUD: Role")
}

func Test_featureFlagCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	backend := api.TestBackend()
	specs.FeatureFlagCrudSpec(api.NewFeatureFlagAPI(&backend))
	RunSpecs(t, "CRUD: Feature Flag")
}

func Test_tenantCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	backend := api.TestBackend()
	specs.TenantCrudSpec(api.NewTenantAPI(&backend))
	RunSpecs(t, "CRUD: Tenant")
}
