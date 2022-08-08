package ssh_access_internals

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_SSHAccessInternals(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ssh_access_internals")
}
