package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/flant/negentropy/cli/cmd/cli/get"
	"github.com/flant/negentropy/cli/cmd/cli/ssh"
	"github.com/flant/negentropy/cli/internal/consts"
)

func main() {
	viper.SetDefault("author", "https://www.flant.com")

	rootCmd := &cobra.Command{
		Use:   "cli",
		Short: "Flant negentropy  CLI", // TODO
		Long: `Flant integration CLI
Find more information at https://flant.com`,
	}
	rootCmd.PersistentFlags().Bool(consts.AllTenantsFlagName, false, "address all tenants of the user: --all-tenants")
	rootCmd.PersistentFlags().Bool(consts.AllProjectsFlagName, false, "address all projects of the user: --all-projects")

	rootCmd.AddCommand(ssh.NewCMD(),
		get.NewCMD())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
