package cmd

import (
	ssh_session "main/pkg/ssh-session"

	"github.com/spf13/cobra"
)

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Create SSH ssh-session to set of servers",
	Long:  ``, // TODO
	Run:   startSSHSession,
}

func init() {
	rootCmd.AddCommand(sshCmd)
}

func startSSHSession(cmd *cobra.Command, args []string) {
	s := ssh_session.Session{}
	s.Go()
}
