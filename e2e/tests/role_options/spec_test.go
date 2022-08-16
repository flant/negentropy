package role_options

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_RoleOptions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create rolebinding with options")
}
