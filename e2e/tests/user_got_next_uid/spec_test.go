package user_got_ssh_access_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_NextUserGotNexUID(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Next_user_got_next_uid")
}
