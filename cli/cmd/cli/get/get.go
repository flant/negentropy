package get

import (
	"github.com/spf13/cobra"

	"github.com/flant/negentropy/cli/internal/consts"
)

func NewCMD() *cobra.Command {
	GetCmd := &cobra.Command{
		Use:   "get",
		Short: "Get information referring available resources",
		Long: `Provide access to referring available resources, 
using: get TYPE_INFO`,
	}
	GetCmd.PersistentFlags().StringP(consts.OutputFlagName, string(consts.OutputFlagName[0]), "",
		"specify output format, available values: [ json | wide | ansible_inventory ]")
	GetCmd.PersistentFlags().Bool(consts.OnlyCacheFlagName, false,
		"specify using only cached data usage: --"+consts.OnlyCacheFlagName)

	GetCmd.AddCommand(TenantCMD(),
		ProjectCMD(),
		ServerCMD())
	return GetCmd
}
