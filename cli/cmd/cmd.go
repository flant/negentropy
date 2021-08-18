package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "flint",
	Short: "Flant integration CLI", // TODO
	Long:  `Flant integration CLI.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
}
