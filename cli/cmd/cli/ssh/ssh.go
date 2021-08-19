package ssh

import (
	"github.com/spf13/cobra"

	"github.com/flant/negentropy/cli/internals/consts"
	session "github.com/flant/negentropy/cli/internals/ssh-session"
	"github.com/flant/negentropy/cli/internals/vault"
)

func NewCMD() *cobra.Command {
	var errSSH error
	SSHCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Create ssh-session to set of servers",
		Long: `Create child bash-session with ability to establish ssh connection with servers specified by flags and params.
To establish connection at child session run 'ssh SERVER_IDENTIFIER@TENANT_IDENTIFIER'`,
		Run: SSHSessionStarter(&errSSH),
		PostRunE: func(command *cobra.Command, args []string) error {
			return errSSH
		},
	}
	SSHCmd.PersistentFlags().Bool(consts.AllTenantsFlagName, false, "address all tenants of the user: --all-tenants")
	SSHCmd.PersistentFlags().StringP(consts.TenantFlagName, string(consts.TenantFlagName[0]), "", "specify one of user tenant: -t first_tenant")
	SSHCmd.PersistentFlags().Bool(consts.AllProjectsFlagName, false, "address all projects of the user: --all=projects")
	SSHCmd.PersistentFlags().StringP(consts.ProjectFlagName, string(consts.ProjectFlagName[0]), "", "specify one of user project at specific tenant: -t first tenant -p main")
	SSHCmd.PersistentFlags().StringArrayP(consts.LabelsFlagName, string(consts.LabelsFlagName[0]), nil, "specify labels of desired servers: --all-tenants -l cloud=aws")
	return SSHCmd
}

func SSHSessionStarter(err *error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()
		params := session.SSHSessionRunParams{Args: args}
		params.AllTenants, *err = flags.GetBool(consts.AllTenantsFlagName)
		if *err != nil {
			return
		}
		params.AllProjects, *err = flags.GetBool(consts.AllProjectsFlagName)
		if *err != nil {
			return
		}
		params.Tenant, *err = flags.GetString(consts.TenantFlagName)
		if *err != nil {
			return
		}
		params.Project, *err = flags.GetString(consts.ProjectFlagName)
		if *err != nil {
			return
		}
		params.Labels, *err = flags.GetStringArray(consts.LabelsFlagName)
		if *err != nil {
			return
		}
		params.Args = args
		*err = params.Validate()
		if *err != nil {
			return
		}

		var s session.Session
		s, *err = session.New(vault.NewService(), params)
		if *err != nil {
			return
		}
		s.Start()
	}
}
