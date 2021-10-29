package project_featureflag

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	pf "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/project_featureflag"
)

func Test_projectFeatureFlagCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	pf.TenantAPI = lib.NewTenantAPI(rootClient)
	pf.FeatureFlagAPI = lib.NewFeatureFlagAPI(rootClient)
	pf.ProjectAPI = lib.NewProjectAPI(rootClient)
	pf.TestAPI = lib.NewProjectFeatureFlagAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Project Feature Flags")
}
