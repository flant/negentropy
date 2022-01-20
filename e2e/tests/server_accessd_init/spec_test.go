package server_accessd_init

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_ServerAccessDInit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "server_accessd_init configure access to test-server")
}
