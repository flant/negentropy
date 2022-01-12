package serviceaccount

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	serviceaccount "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/service_account"
)

func Test_userCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	serviceaccount.TenantAPI = lib.NewTenantAPI(rootClient)
	serviceaccount.TestAPI = lib.NewServiceAccountAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: ServiceAccount")
}
