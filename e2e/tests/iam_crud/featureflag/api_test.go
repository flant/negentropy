package featureflag

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs/featureflag"
)

func Test_featureFlagCRUD(t *testing.T) {
	rootClient := lib.NewConfiguredIamVaultClient()
	featureflag.TestAPI = lib.NewFeatureFlagAPI(rootClient)
	RegisterFailHandler(Fail)
	RunSpecs(t, "CRUD: FeatureFlag")
}
