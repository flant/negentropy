package project

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/project"
)

func Test_projectCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	project.TenantAPI = lib.NewTenantAPI(rootClient)
	project.TestAPI = lib.NewProjectAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Project")
}
