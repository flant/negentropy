package team

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/api"
)

func Test_teamCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewTeamAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Team")
}
