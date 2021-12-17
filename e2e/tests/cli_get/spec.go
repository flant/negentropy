package cli_get

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/e2e/tests/lib/test_server_and_client_preparing"
)

var testServerAndClientSuite test_server_and_client_preparing.Suite

var flantIamSuite flant_iam_preparing.Suite

type Cfg = flant_iam_preparing.CheckingEnvironment

var cfg Cfg

var _ = BeforeSuite(func() {
	testServerAndClientSuite.BeforeSuite()
	flantIamSuite.BeforeSuite()
	Describe("configuring system", func() {
		cfg = flantIamSuite.PrepareForSSHTesting()
		time.Sleep(time.Second * 10)
		testServerAndClientSuite.PrepareClientForSSHTesting(cfg)
	})
}, 1.0)

var _ = Describe("Process of running cli get", func() {
	Describe("cli get tenant", func() {
		DescribeTable("cli get tenant command variations",
			func(runningCliGetCmdBuild func(cfg Cfg) string) {
				d := testServerAndClientSuite
				runningCliCmd := runningCliGetCmdBuild(cfg)
				output := d.ExecuteCommandAtContainer(d.TestClientContainer, []string{
					"/bin/bash", "-c",
					runningCliCmd,
				},
					[]string{})
				tmp := strings.Join(output, "")

				Expect(tmp).To(ContainSubstring("output flag="))
				Expect(tmp).To(ContainSubstring("tenants:"))
				Expect(tmp).To(ContainSubstring(cfg.Tenant.Identifier))
			},
			Entry("/opt/cli/bin/cli get tenant",
				func(Cfg) string {
					return "/opt/cli/bin/cli get tenant"
				}),

			Entry("/opt/cli/bin/cli get tenant XXX",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli  get tenant %s\n", cfg.Tenant.Identifier)
				}),
		)
	})
	Describe("cli get project", func() {
		DescribeTable("cli get project command variations",
			func(runningCliGetCmdBuild func(cfg Cfg) string) {
				d := testServerAndClientSuite
				runningCliCmd := runningCliGetCmdBuild(cfg)
				output := d.ExecuteCommandAtContainer(d.TestClientContainer, []string{
					"/bin/bash", "-c",
					runningCliCmd,
				},
					[]string{})
				tmp := strings.Join(output, "")

				Expect(tmp).To(ContainSubstring("output flag="))
				Expect(tmp).To(ContainSubstring("projects:"))
				Expect(tmp).To(ContainSubstring(cfg.Project.Identifier))
			},
			Entry("/opt/cli/bin/cli get project",
				func(Cfg) string {
					return "/opt/cli/bin/cli get project"
				}),

			Entry("/opt/cli/bin/cli -t XXX get project",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli -t %s get project \n", cfg.Tenant.Identifier)
				}),
			Entry("/opt/cli/bin/cli -t XXX get project YYY",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli -t %s get project %s \n", cfg.Tenant.Identifier, cfg.Project.Identifier)
				}),
			Entry("/opt/cli/bin/cli -t XXX get project YYY ZZZ",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli -t %s get project %s FAKE_PROJECT_ID \n", cfg.Tenant.Identifier, cfg.Project.Identifier)
				}),

			Entry("/opt/cli/bin/cli --all-tenants get project",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli --all-tenants get project \n") // It works
				}),
		)
	})
	Describe("cli get server", func() {
		DescribeTable("cli get serer command variations",
			func(runningCliGetCmdBuild func(cfg Cfg) string) {
				d := testServerAndClientSuite
				runningCliCmd := runningCliGetCmdBuild(cfg)
				output := d.ExecuteCommandAtContainer(d.TestClientContainer, []string{
					"/bin/bash", "-c",
					runningCliCmd,
				},
					[]string{})
				tmp := strings.Join(output, "")

				Expect(tmp).To(ContainSubstring("output flag="))
				Expect(tmp).To(ContainSubstring("servers:"))
				Expect(tmp).To(ContainSubstring(cfg.Tenant.Identifier + "." + cfg.TestServerIdentifier))
			},
			Entry("/opt/cli/bin/cli get server --all-tenants",
				func(Cfg) string {
					return "/opt/cli/bin/cli get server --all-tenants"
				}),

			Entry("/opt/cli/bin/cli -t 1tv -p main get server",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli -t %s -p %s get server \n", cfg.Tenant.Identifier, cfg.Project.Identifier)
				}),

			Entry("/opt/cli/bin/cli -t 1tv -p main get server db-1",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli -t %s -p %s get server %s \n", cfg.Tenant.Identifier, cfg.Project.Identifier, cfg.TestServerIdentifier)
				}),

			Entry("/opt/cli/bin/cli -t 1tv -p main get server db-1 db-2",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli -t %s -p %s get server %s FAKE_SERVER_ID \n", cfg.Tenant.Identifier, cfg.Project.Identifier, cfg.TestServerIdentifier)
				}),

			Entry("/opt/cli/bin/cli -t 1tv -p main get server -l XXX",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli -t %s -p %s get server -l %s  \n", cfg.Tenant.Identifier, cfg.Project.Identifier, labelSelector(cfg))
				}),

			Entry("/opt/cli/bin/cli  -t 1tv --all-projects get server",
				func(cfg Cfg) string {
					// return fmt.Sprintf("/opt/cli/bin/cli -t %s --all-projects get server \n", cfg.Tenant.Identifier ) // it  doesn't work
					return fmt.Sprintf("/opt/cli/bin/cli -t %s  get server --all-projects \n", cfg.Tenant.Identifier)
				}),
			Entry("/opt/cli/bin/cli  -t 1tv --all-projects get server -l XXX",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli -t %s --all-projects get server -l %s  \n", cfg.Tenant.Identifier, labelSelector(cfg)) // It works
				}),
			Entry("/opt/cli/bin/cli  --all-tenants get server",
				func(cfg Cfg) string {
					// return fmt.Sprintf("/opt/cli/bin/cli --all-tenants get server  \n") // It doesn't work
					return fmt.Sprintf("/opt/cli/bin/cli  get server --all-tenants \n")
				}),
			Entry("/opt/cli/bin/cli  --all-tenants get server -l -l XXX",
				func(cfg Cfg) string {
					return fmt.Sprintf("/opt/cli/bin/cli --all-tenants get server -l %s  \n", labelSelector(cfg)) // It  works
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

func labelSelector(cfg Cfg) string {
	var result string
	for k, v := range cfg.ServerLabels {
		result = k + "=" + v
		break // just one pair
	}
	return result
}
