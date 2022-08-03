package user_got_ssh_access_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func T_est_UserSSHAccess(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User_got_ssh_access")
}
