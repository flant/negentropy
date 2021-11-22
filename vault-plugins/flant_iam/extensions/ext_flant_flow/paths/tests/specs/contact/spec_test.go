package contact

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testapi "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/api"
)

func Test_contactCRUD(t *testing.T) {
	backend := testapi.TestBackend()
	TestAPI = api.NewContactAPI(&backend)
	ClientAPI = api.NewClientAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Contact")
}
