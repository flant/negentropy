package teammate_got_ssh_access_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_TeammateSSHAccess(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Teammate_got_ssh_access")
}
