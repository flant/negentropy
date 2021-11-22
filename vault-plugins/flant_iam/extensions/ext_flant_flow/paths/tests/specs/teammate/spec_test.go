package teammate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/api"
)

func Test_teammateCRUD(t *testing.T) {
	backend := testapi.TestBackend()
	TestAPI = api.NewTeammateAPI(&backend)
	TeamAPI = api.NewTeamAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Teammate")
}
