package identitysharing

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_identitySharingCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	backend := api.TestBackend()
	TenantAPI = api.NewTenantAPI(&backend)
	TestAPI = api.NewIdentitySharingAPI(&backend)
	RunSpecs(t, "CRUD flant_iam: IdentitySharing")
}
