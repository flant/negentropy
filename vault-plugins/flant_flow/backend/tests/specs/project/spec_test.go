package project

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/api"
)

func Test_projectCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewProjectAPI(&backend)
	ClientAPI = api.NewClientAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Project")
}
