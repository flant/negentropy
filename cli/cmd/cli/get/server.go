package get

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flant/negentropy/cli/internal/consts"
	"github.com/flant/negentropy/cli/internal/model"
	"github.com/flant/negentropy/cli/internal/vault"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func ServerCMD() *cobra.Command {
	var getServerErr error
	getServerCmd := &cobra.Command{
		Use:   "server",
		Short: "Get server information",
		Long: `Provide access to server information,
using: get -flags server [PROJECT_IDENTIFIERS]`,
		Run: server(&getServerErr),
		PostRunE: func(command *cobra.Command, args []string) error {
			return getServerErr
		},
	}
	getServerCmd.PersistentFlags().Bool(consts.AllTenantsFlagName, false, "address all tenants of the user: --all-tenants")
	getServerCmd.PersistentFlags().StringP(consts.TenantFlagName, string(consts.TenantFlagName[0]), "", "specify one of user tenant: -t first_tenant")
	getServerCmd.PersistentFlags().Bool(consts.AllProjectsFlagName, false, "address all projects of the user: --all-projects")
	getServerCmd.PersistentFlags().StringP(consts.ProjectFlagName, string(consts.ProjectFlagName[0]), "", "specify one of user project at specific tenant: -t first tenant -p main")
	getServerCmd.PersistentFlags().StringP(consts.LabelsFlagName, string(consts.LabelsFlagName[0]), "", "specify labels of desired servers: --all-tenants -l cloud=aws")

	return getServerCmd
}

func server(outErr *error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()

		serverFilter := model.ServerFilter{}
		serverFilter.AllTenants, *outErr = flags.GetBool(consts.AllTenantsFlagName)
		if *outErr != nil {
			return
		}
		serverFilter.AllProjects, *outErr = flags.GetBool(consts.AllProjectsFlagName)
		if *outErr != nil {
			return
		}
		serverFilter.TenantIdentifiers, *outErr = model.StringSetFromStringFlag(flags, consts.TenantFlagName)
		if *outErr != nil {
			return
		}
		serverFilter.ProjectIdentifiers, *outErr = model.StringSetFromStringFlag(flags, consts.ProjectFlagName)
		if *outErr != nil {
			return
		}
		serverFilter.LabelSelector, *outErr = flags.GetString(consts.LabelsFlagName)
		if *outErr != nil {
			return
		}
		serverFilter.ServerIdentifiers = args

		if len(serverFilter.ServerIdentifiers) == 0 {
			serverFilter.AllServers = true
		}

		*outErr = serverFilter.Validate()
		if *outErr != nil {
			return
		}

		var cache *model.ServerList
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

		serverList, err := getServerData(onlyCache, cache, serverFilter, permanentCacheFilePath)
		if err != nil {
			*outErr = err
			return
		}

		fmt.Printf("output flag= %s\n", output)
		fmt.Printf("servers: \n")
		for _, s := range serverList.Servers {
			fmt.Printf("%s.%s\n", serverList.Tenants[s.TenantUUID].Identifier, s.Identifier)
		}

		fmt.Printf("========\n")
	}
}

func getServerData(onlyCache bool, cache *model.ServerList, serverFilter model.ServerFilter,
	permanentCacheFilePath string) (*model.ServerList, error) {
	var (
		tenants  map[iam.TenantUUID]iam.Tenant
		projects map[iam.ProjectUUID]iam.Project
		servers  map[ext.ServerUUID]ext.Server
	)

	var serverList *model.ServerList

	if onlyCache {
		tenants = map[iam.TenantUUID]iam.Tenant{}
		for uuid, t := range cache.Tenants {
			if serverFilter.TenantIdentifiers.IsEmpty() || serverFilter.TenantIdentifiers.Contains(t.Identifier) {
				tenants[uuid] = t
			}
		}
		projects = map[iam.ProjectUUID]iam.Project{}
		for uuid, p := range cache.Projects {
			if serverFilter.ProjectIdentifiers.IsEmpty() || serverFilter.ProjectIdentifiers.Contains(p.Identifier) {
				projects[uuid] = p
			}
		}
		servers = map[ext.ServerUUID]ext.Server{}
		for uuid, s := range cache.Servers {
			idSet := model.StringSet{}
			for _, id := range serverFilter.ServerIdentifiers {
				idSet.Put(id)
			}
			if idSet.IsEmpty() || idSet.Contains(s.Identifier) {
				servers[uuid] = s
			}
		}
		serverList = &model.ServerList{
			Tenants:  tenants,
			Projects: projects,
			Servers:  servers,
		}
	} else {
		var err error
		serverList, err = vault.NewService().UpdateServersByFilter(serverFilter, cache)
		if err != nil {
			return nil, err
		}
		*cache = model.UpdateServerListCacheWithFreshValues(*cache, *serverList)

		model.SaveToFile(*cache, permanentCacheFilePath)
	}
	return serverList, nil
}
