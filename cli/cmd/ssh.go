package cmd

import (
	"main/pkg/session"

	"github.com/spf13/cobra"
)

var (
	sshCmd = &cobra.Command{
		Use:   "ssh",
		Short: "Create SSH session to set of servers",
		Long:  ``, // TODO
		Run:   startSSHSession,
	}
)

func init() {
	command.AddCommand(sshCmd)
}

func startSSHSession(cmd *cobra.Command, args []string) {
	s := session.Session{}
	s.Go()
}
