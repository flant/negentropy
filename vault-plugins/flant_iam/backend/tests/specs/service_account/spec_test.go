package serviceaccount

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_serviceaccountCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewServiceAccountAPI(&backend)
	TenantAPI = api.NewTenantAPI(&backend)
	IdentitySharingAPI = api.NewIdentitySharingAPI(&backend)
	GroupAPI = api.NewGroupAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_iam: ServiceAccount")
}
