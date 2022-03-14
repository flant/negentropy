package project

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/backend/tests/specs/policy"
)

func Test_projectCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	authClient := lib.NewConfiguredIamAuthVaultClient()

	policy.TestAPI = lib.NewPolicyAPI(authClient)
	policy.RoleAPI = lib.NewRoleAPI(rootClient)

	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD flant_iam_auth: Policy")
}
