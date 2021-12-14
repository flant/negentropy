package access_token_auth

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_AccessTokenAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User use access_token issued by oidc provider")
}
