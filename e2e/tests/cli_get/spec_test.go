package cli_get

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_CliGet(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User_run_cli_get_commands")
}
