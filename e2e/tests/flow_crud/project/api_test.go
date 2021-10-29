package project

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/specs/project"
)

func Test_projectCRUD(t *testing.T) {
	flowRootClient := lib.NewConfiguredFlowRootVaultClient()
	project.ClientAPI = lib.NewFlowClientAPI(flowRootClient)
	project.TestAPI = lib.NewFlowProjectClientAPI(flowRootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Project")
}
