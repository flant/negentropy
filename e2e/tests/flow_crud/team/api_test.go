package team

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs/team"
)

func Test_teamCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	team.TestAPI = lib.NewFlowTeamAPI(rootClient)
	team.RoleAPI = lib.NewRoleAPI(rootClient)
	team.ConfigAPI = lib.NewHttpClientBasedConfigAPI(rootClient)

	team.GroupAPI = lib.NewGroupAPI(rootClient)
	team.TeammateAPI = lib.NewFlowTeammateAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_flow: Team")
}
