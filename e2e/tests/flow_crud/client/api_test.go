package client

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/specs/client"
)

func Test_clientCRUD(t *testing.T) {
	flowRootClient := lib.NewConfiguredFlowRootVaultClient()
	client.TestAPI = lib.NewFlowClientAPI(flowRootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_flow: Client")
}
