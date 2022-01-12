package serviceaccountpassword

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	serviceaccountpassword "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/service_account_password"
)

func Test_serviceaccountpasswordCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	serviceaccountpassword.TenantAPI = lib.NewTenantAPI(rootClient)
	serviceaccountpassword.ServiceAccountAPI = lib.NewServiceAccountAPI(rootClient)
	serviceaccountpassword.TestAPI = lib.NewServiceAccountPasswordAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: ServiceAccount password")
}
