package teammate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs/teammate"
)

func Test_teammateCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	teammate.TeamAPI = lib.NewFlowTeamAPI(rootClient)
	teammate.TestAPI = lib.NewFlowTeammateAPI(rootClient)
	teammate.TenantAPI = lib.NewTenantAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_flow: Teammate")
}
