package team

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/api"
)

func Test_teamCRUD(t *testing.T) {
	backend := testapi.TestBackend()
	TestAPI = api.NewTeamAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Team")
}
