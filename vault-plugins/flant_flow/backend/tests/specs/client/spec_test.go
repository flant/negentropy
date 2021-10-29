package client

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_flow/backend/tests/api"
)

func Test_clientCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewClientAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Tenant")
}
