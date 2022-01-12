package serviceaccountmultipass

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_serviceAccountMultipassCRUD(t *testing.T) {
	backend, storage := api.TestBackendWithStorage()
	ServiceAccountAPI = api.NewServiceAccountAPI(&backend)
	TenantAPI = api.NewTenantAPI(&backend)
	TestAPI = api.NewServiceAccountMultipassAPI(&backend)
	ConfigAPI = api.NewBackendBasedConfigAPI(&backend, &storage)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_iam: ServiceAccount Multipass")
}
