package access_token_or_sapass_auth

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_RenewVST(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User use oidc provider and jwt, and then renew vst")
}
