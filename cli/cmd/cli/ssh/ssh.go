package ssh

import (
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/flant/negentropy/cli/internal/consts"
	"github.com/flant/negentropy/cli/internal/model"
	session "github.com/flant/negentropy/cli/internal/ssh-session"
	"github.com/flant/negentropy/cli/internal/vault"
)

func NewCMD() *cobra.Command {
	var errSSH error
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Create ssh-session to set of servers",
		Long: `Create child bash-session with ability to establish ssh connection with servers specified by flags and params.
To establish connection at child session run 'ssh SERVER_IDENTIFIER@TENANT_IDENTIFIER'`,
		Run: SSHSessionStarter(&errSSH),
		PostRunE: func(command *cobra.Command, args []string) error {
			return errSSH
		},
	}
	sshCmd.PersistentFlags().Bool(consts.AllTenantsFlagName, false, "address all tenants of the user: --all-tenants")
	sshCmd.PersistentFlags().StringP(consts.TenantFlagName, string(consts.TenantFlagName[0]), "", "specify one of user tenant: -t first_tenant")
	sshCmd.PersistentFlags().Bool(consts.AllProjectsFlagName, false, "address all projects of the user: --all-projects")
	sshCmd.PersistentFlags().StringP(consts.ProjectFlagName, string(consts.ProjectFlagName[0]), "", "specify one of user project at specific tenant: -t first tenant -p main")
	sshCmd.PersistentFlags().StringP(consts.LabelsFlagName, string(consts.LabelsFlagName[0]), "", "specify labels of desired servers: --all-tenants -l cloud=aws")
	sshCmd.PersistentFlags().Bool(consts.AllServersFlagName, false, "address all servers: --all")

	return sshCmd
}

func SSHSessionStarter(err *error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		serverFilter := model.ServerFilter{}
		serverFilter.AllTenants, *err = flags.GetBool(consts.AllTenantsFlagName)
		if *err != nil {
			return
		}
		serverFilter.AllProjects, *err = flags.GetBool(consts.AllProjectsFlagName)
		if *err != nil {
			return
		}
		serverFilter.TenantIdentifiers, *err = model.StringSetFromStringFlag(flags, consts.TenantFlagName)
		if *err != nil {
			return
		}
		serverFilter.ProjectIdentifiers, *err = model.StringSetFromStringFlag(flags, consts.ProjectFlagName)
		if *err != nil {
			return
		}
		serverFilter.LabelSelector, *err = flags.GetString(consts.LabelsFlagName)
		if *err != nil {
			return
		}
		serverFilter.AllServers, *err = flags.GetBool(consts.AllServersFlagName)
		if *err != nil {
			return
		}
		serverFilter.ServerIdentifiers = args

		if len(serverFilter.ServerIdentifiers) == 0 {
			serverFilter.AllServers = true
		}

		*err = serverFilter.Validate()
		if *err != nil {
			return
		}

		var (
			s       *session.Session
			homeDir string
		)
		homeDir, *err = os.UserHomeDir()
		if *err != nil {
			return
		}
		permanentCacheFilePath := path.Join(homeDir, ".flant", "cli", "ssh", "cache")

		s, *err = session.New(vault.NewService(), serverFilter, permanentCacheFilePath)
		if *err != nil {
			return
		}
		*err = s.Start()
	}
}
