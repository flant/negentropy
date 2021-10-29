package teammate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/specs/teammate"
)

func Test_teammateCRUD(t *testing.T) {
	flowRootClient := lib.NewConfiguredFlowRootVaultClient()
	teammate.TeamAPI = lib.NewFlowTeamAPI(flowRootClient)
	teammate.TestAPI = lib.NewFlowTeammateAPI(flowRootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Teammate")
}
