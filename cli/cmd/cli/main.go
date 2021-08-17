package main

import (
	"fmt"
	"os"

	"github.com/flant/negentropy/cli/cmd/cli/ssh"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

func main() {
	viper.SetDefault("author", "https://www.flant.com")

	rootCmd := &cobra.Command{
		Use:   "cli",
		Short: "Flant negentropy  CLI", // TODO
		Long: `Flant integration CLI
Find more information at https://flant.com`,
	}

	rootCmd.AddCommand(ssh.NewCMD())
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
