package usermultipass

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/e2e/tests/lib"
	usermultipass "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/user_multipass"
)

func Test_tenantCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	usermultipass.TenantAPI = lib.NewTenantAPI(rootClient)
	usermultipass.UserAPI = lib.NewUserAPI(rootClient)
	usermultipass.TestAPI = lib.NewUserMultipassAPI(rootClient)
	usermultipass.ConfigAPI = lib.NewHttpClientBasedConfigAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: User Multipass")
}
