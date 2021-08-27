package get

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/flant/negentropy/cli/internal/consts"
	"github.com/flant/negentropy/cli/internal/model"
	"github.com/flant/negentropy/cli/internal/vault"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func TenantCMD() *cobra.Command {
	var getTenantErr error
	getTenantCmd := &cobra.Command{
		Use:   "tenant",
		Short: "Get tenant information",
		Long: `Provide access to referring available yenany information,
using: get tenant [TENANT_IDENTIFIERS]`,
		Run: tenant(&getTenantErr),
		PostRunE: func(command *cobra.Command, args []string) error {
			return getTenantErr
		},
	}

	return getTenantCmd
}

func tenant(outErr *error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()

		serverFilter := model.ServerFilter{}

		if len(args) == 0 {
			serverFilter.AllTenants = true
			serverFilter.TenantIdentifiers = model.StringSet{}
		} else {
			set := model.StringSet{}
			for _, ti := range args {
				set.Put(ti)
			}
			serverFilter.TenantIdentifiers = set
		}

		cache, permanentCacheFilePath, err := readCache()
		if err != nil {
			*outErr = err
			return
		}

		onlyCache, output, err := getOutputAndCacheFlags(flags)
		if err != nil {
			*outErr = err
			return
		}

		tenants, err := getTenantData(onlyCache, cache, serverFilter, permanentCacheFilePath)
		if err != nil {
			*outErr = err
			return
		}

		fmt.Printf("output flag = %s\n", output)
		fmt.Printf("tenants: \n")
		for _, t := range tenants {
			fmt.Printf("%s\n", t.Identifier)
		}
		fmt.Printf("========\n")
	}
}

func getTenantData(onlyCache bool, cache *model.ServerList, serverFilter model.ServerFilter,
	permanentCacheFilePath string) (map[iam.TenantUUID]iam.Tenant, error) {
	var tenants map[iam.TenantUUID]iam.Tenant

	if onlyCache {
		tenants = map[iam.TenantUUID]iam.Tenant{}
		for uuid, t := range cache.Tenants {
			if serverFilter.AllTenants || serverFilter.TenantIdentifiers.Contains(t.Identifier) {
				tenants[uuid] = t
			}
		}
	} else {
		var err error
		tenants, err = vault.NewService().UpdateTenants(cache.Tenants, serverFilter.TenantIdentifiers)

		if err != nil {
			return nil, err
		}
		*cache = model.UpdateServerListCacheWithFreshValues(*cache, model.ServerList{
			Tenants: tenants,
		})
		model.SaveToFile(*cache, permanentCacheFilePath)
	}
	return tenants, nil
}

func getOutputAndCacheFlags(flags *pflag.FlagSet) (bool, string, error) {
	onlyCache, err := flags.GetBool(consts.OnlyCacheFlagName)
	if err != nil {
		return false, "", err
	}
	output, err := flags.GetString(consts.OutputFlagName)
	if err != nil {
		return false, "", err
	}
	return onlyCache, output, nil
}

func readCache() (*model.ServerList, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", err
	}
	permanentCacheFilePath := path.Join(homeDir, ".flant", "cli", "ssh", "cache")

	cache, err := model.TryReadCacheFromFile(permanentCacheFilePath)
	if err != nil {
		err = fmt.Errorf("get project, reading permanent cache: %w", err)
		return nil, "", err
	}
	return cache, permanentCacheFilePath, nil
}
