package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	command = &cobra.Command{
		Use:   "flint",
		Short: "Flant integration CLI", // TODO
		Long:  `Flant integration CLI.`,
	}
)

func Execute() error {
	return command.Execute()
}

func init() {
	viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
}
