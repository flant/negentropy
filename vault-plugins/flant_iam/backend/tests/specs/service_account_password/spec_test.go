package serviceaccountpassword

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_serviceaccountpassswordCRUD(t *testing.T) {
	backend := api.TestBackend()
	TestAPI = api.NewServiceAccountPasswordAPI(&backend)
	ServiceAccountAPI = api.NewServiceAccountAPI(&backend)
	TenantAPI = api.NewTenantAPI(&backend)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_iam: ServiceAccount")
}
