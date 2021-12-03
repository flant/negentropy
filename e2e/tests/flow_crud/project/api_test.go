package project

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs/project"
)

func Test_projectCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	project.ClientAPI = lib.NewFlowClientAPI(rootClient)
	project.TestAPI = lib.NewFlowProjectAPI(rootClient)
	project.TenantAPI = lib.NewTenantAPI(rootClient)
	project.RoleAPI = lib.NewRoleAPI(rootClient)
	project.TeamAPI = lib.NewFlowTeamAPI(rootClient)
	project.ConfigAPI = lib.NewHttpClientBasedConfigAPI(rootClient)

	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_flow: Project")
}
