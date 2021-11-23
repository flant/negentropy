package teammate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs/contact"
)

func Test_teammateCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	contact.ClientAPI = lib.NewFlowClientAPI(rootClient)
	contact.TestAPI = lib.NewFlowContactAPI(rootClient)
	contact.ProjectAPI = lib.NewFlowProjectAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_flow: Contact")
}
