package user_got_ssh_access

import (
	_ "embed"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/e2e/tests/lib/test_server_and_client_preparing"
)

var testServerAndClientSuite test_server_and_client_preparing.Suite

var flantIamSuite flant_iam_preparing.Suite

var cfg flant_iam_preparing.CheckingEnvironment

var _ = BeforeSuite(func() {
	testServerAndClientSuite.BeforeSuite()
	flantIamSuite.BeforeSuite()
	Describe("configuring system", func() {
		cfg = flantIamSuite.PrepareForSSHTesting()
		err := flantIamSuite.WaitPrepareForSSHTesting(cfg, 40)
		Expect(err).ToNot(HaveOccurred())
		testServerAndClientSuite.PrepareServerForSSHTesting(cfg)

		testServerAndClientSuite.PrepareClientForSSHTesting(cfg)
	})
}, 1.0)

var _ = Describe("Process of getting ssh access to server by a user", func() {
	Describe("cli ssh running", func() {
		testCounter := 1
		DescribeTable("cli ssh command variations",
			func(buildCliCmd func(cfg flant_iam_preparing.CheckingEnvironment) string) {
				d := testServerAndClientSuite
				// Expect(d.DirectoryAtContainerNotExistOrEmpty(d.TestClientContainer, "/tmp/flint")).To(BeTrue(),
				//	"/tmp/flint files doesn't exist before start")

				runningCliCmd := buildCliCmd(cfg)
				sshCmd := fmt.Sprintf("ssh -oStrictHostKeyChecking=accept-new %s.%s", cfg.Project.Identifier,
					cfg.TestServerIdentifier)
				testFilePath := fmt.Sprintf("/home/%s/test%d.txt", cfg.User.Identifier, testCounter)
				touchCommand := "touch " + testFilePath
				fmt.Printf("\n%s\n", runningCliCmd)
				fmt.Printf("\n%s\n", sshCmd)
				fmt.Printf("\n%s\n", touchCommand)
				output := d.ExecuteCommandAtContainer(d.TestClientContainer, []string{
					"/bin/bash", "-c",
					runningCliCmd,
				},
					[]string{
						sshCmd,
						touchCommand,
						"exit", "exit",
					})

				writeLogToFile(output, fmt.Sprintf("cli%d.log", testCounter))
				testCounter++

				Expect(d.DirectoryAtContainerNotExistOrEmpty(d.TestClientContainer, "/tmp/flint")).To(BeTrue(),
					"/tmp/flint files doesn't exist after closing cli")

				Expect(d.CheckFileExistAtContainer(d.TestServerContainer, testFilePath, "f")).
					ToNot(HaveOccurred(), "after run cli ssh - test file is created at server")

				Expect(d.CheckFileExistAtContainer(d.TestClientContainer, "/tmp/flint", "d")).
					ToNot(HaveOccurred(), "after run cli ssh - tmp dir exists at client container")

				Expect(d.DirectoryAtContainerNotExistOrEmpty(d.TestClientContainer, "/tmp/flint")).To(BeTrue(),
					"/tmp/flint is empty  after closing cli")
			},
			Entry("[-t XXX --all-projects]",
				func(cfg flant_iam_preparing.CheckingEnvironment) string {
					return fmt.Sprintf("/opt/cli/bin/cli  ssh -t %s --all-projects\n", cfg.Tenant.Identifier)
				}),
			Entry("[--all-tenants --all-projects -l system=ubuntu20]",
				func(cfg flant_iam_preparing.CheckingEnvironment) string {
					var labelSelector string
					for k, v := range cfg.ServerLabels {
						labelSelector = k + "=" + v
						break // just one pair
					}
					return fmt.Sprintf("/opt/cli/bin/cli  ssh --all-tenants --all-projects -l %s\n", labelSelector)
				}),
			Entry("[--all-tenants -p XXX  test-server]",
				func(cfg flant_iam_preparing.CheckingEnvironment) string {
					return fmt.Sprintf("/opt/cli/bin/cli  ssh --all-tenants -p %s %s\n", cfg.Project.Identifier,
						cfg.TestServerIdentifier)
				}),
			Entry(" [-t XXX -p YYY]",
				func(cfg flant_iam_preparing.CheckingEnvironment) string {
					return fmt.Sprintf("/opt/cli/bin/cli  ssh -t %s -p %s\n", cfg.Tenant.Identifier, cfg.Project.Identifier)
				}),
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
