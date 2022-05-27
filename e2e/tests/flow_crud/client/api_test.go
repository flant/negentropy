package client

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs/client"
)

func Test_clientCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	client.TestAPI = lib.NewFlowClientAPI(rootClient)
	client.RoleAPI = lib.NewRoleAPI(rootClient)
	client.TeamAPI = lib.NewFlowTeamAPI(rootClient)
	client.GroupAPI = lib.NewGroupAPI(rootClient)
	client.ConfigAPI = lib.NewHttpClientBasedConfigAPI(rootClient)
	client.IdentitySharingAPI = lib.NewIdentitySharingAPI(rootClient)
	client.UserAPI = lib.NewUserAPI(rootClient)
	client.RoleBindingAPI = lib.NewRoleBindingAPI(rootClient)

	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_flow: Client")
}
