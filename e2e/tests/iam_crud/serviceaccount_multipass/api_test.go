package serviceaccountmulipass

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	serviceaccountmulipass "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/service_account_multipass"
)

func Test_serviceaccountmulipassCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	serviceaccountmulipass.TenantAPI = lib.NewTenantAPI(rootClient)
	serviceaccountmulipass.ServiceAccountAPI = lib.NewServiceAccountAPI(rootClient)
	serviceaccountmulipass.TestAPI = lib.NewServiceAccountMultipassAPI(rootClient)
	serviceaccountmulipass.ConfigAPI = lib.NewHttpClientBasedConfigAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: ServiceAccount multipass")
}
