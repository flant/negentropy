package teammate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/specs/contact"
)

func Test_teammateCRUD(t *testing.T) {
	flowRootClient := lib.NewConfiguredFlowRootVaultClient()
	contact.ClientAPI = lib.NewFlowClientAPI(flowRootClient)
	contact.TestAPI = lib.NewFlowContactAPI(flowRootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_flow: Contact")
}
