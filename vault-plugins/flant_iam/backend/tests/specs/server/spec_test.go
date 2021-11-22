package server

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_serverCRUD(t *testing.T) {
	backend, storage := api.TestBackendWithStorage()
	TestAPI = api.NewServerAPI(&backend, &storage)
	TenantAPI = api.NewTenantAPI(&backend)
	ProjectAPI = api.NewProjectAPI(&backend)
	RoleAPI = api.NewRoleAPI(&backend)
	ConfigAPI = api.NewBackendBasedConfigAPI(&backend, &storage)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_iam: Server")
}
