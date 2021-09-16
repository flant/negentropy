package project_featureflag

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_projectFeatureFlagCRUD(t *testing.T) {
	backend := api.TestBackend()
	TenantAPI = api.NewTenantAPI(&backend)
	ProjectAPI = api.NewProjectAPI(&backend)
	FeatureFlagAPI = api.NewFeatureFlagAPI(&backend)
	TestAPI = api.NewProjectFeatureFlagAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Project Feature Flags")
}
