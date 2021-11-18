package contact

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/backend/tests/api"
)

func Test_contactCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewContactAPI(&backend)
	ClientAPI = api.NewClientAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Contact")
}
