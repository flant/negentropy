package featureflag

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
)

func Test_featureFlagCRUD(t *testing.T) {
	RegisterFailHandler(Fail)
	backend := api.TestBackend()
	TestAPI = api.NewFeatureFlagAPI(&backend)
	RunSpecs(t, "CRUD: Feature Flag")
}
