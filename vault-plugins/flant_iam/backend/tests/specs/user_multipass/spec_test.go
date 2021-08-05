package usermultipass

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_userMultipassCRUD(t *testing.T) {
	backend, storage := api.TestBackendWithStorage()
	UserAPI = api.NewUserAPI(&backend)
	TenantAPI = api.NewTenantAPI(&backend)
	TestAPI = api.NewUserMultipassAPI(&backend)
	ConfigAPI = api.NewBackendBasedConfigAPI(&backend, &storage)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: User Multipass")
}
