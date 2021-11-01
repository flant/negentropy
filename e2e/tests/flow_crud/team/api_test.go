package team

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/specs/team"
)

func Test_teamCRUD(t *testing.T) {
	flowRootClient := lib.NewConfiguredFlowRootVaultClient()
	team.TestAPI = lib.NewFlowTeamAPI(flowRootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_flow: Team")
}
