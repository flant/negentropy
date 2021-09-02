package cli_get

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/e2e/tests/lib/test_server_and_client_preparing"
)

var testServerAndClientSuite test_server_and_client_preparing.Suite

var flantIamSuite flant_iam_preparing.Suite

var cfg flant_iam_preparing.CheckingSSHConnectionEnvironment

var _ = BeforeSuite(func() {
	testServerAndClientSuite.BeforeSuite()
	flantIamSuite.BeforeSuite()
	Describe("configuring system", func() {
		cfg = flantIamSuite.PrepareForSSHTesting()

		testServerAndClientSuite.PrepareClientForSSHTesting(cfg)
	})
}, 1.0)

var _ = Describe("Process of running cli get", func() {
	Describe("cli ssh running", func() {
		testCounter := 1
		DescribeTable("cli ssh command variations",
			func(runningCliCmd string) {
				d := testServerAndClientSuite

				fmt.Printf("\n%s\n", runningCliCmd)
				output := d.ExecuteCommandAtContainer(d.TestClientContainer, []string{
					"/bin/bash", "-c",
					runningCliCmd,
				},
					[]string{})

				fmt.Println(output)
				// writeLogToFile(output, fmt.Sprintf("cli%d.log", testCounter))
				testCounter++

				tmp := strings.Join(output, "")

				Expect(tmp).To(ContainSubstring("output flag ="))
				Expect(tmp).To(ContainSubstring("tenants:"))
				Expect(tmp).To(ContainSubstring("tenants:"))
			},
			Entry("/opt/cli/bin/cli get tenant", "/opt/cli/bin/cli get tenant"),
		)
	})
})

func writeLogToFile(output []string, logFilePath string) {
	logFile, err := os.Create(logFilePath)
	if err != nil {
		panic(err)
	}
	for _, s := range output {
		logFile.WriteString(s)
	}
	logFile.Close()
}
