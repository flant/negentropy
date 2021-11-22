package server

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/server"
)

func Test_serverCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	server.TenantAPI = lib.NewTenantAPI(rootClient)
	server.ProjectAPI = lib.NewProjectAPI(rootClient)
	server.RoleAPI = lib.NewRoleAPI(rootClient)
	server.TestAPI = lib.NewServerAPI(rootClient)
	server.ConfigAPI = lib.NewHttpClientBasedConfigAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: Server")
}
