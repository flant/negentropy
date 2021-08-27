package get

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/flant/negentropy/cli/internal/consts"
	"github.com/flant/negentropy/cli/internal/model"
	"github.com/flant/negentropy/cli/internal/vault"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func ProjectCMD() *cobra.Command {
	var getProjectErr error
	getProjectCmd := &cobra.Command{
		Use:   "project",
		Short: "Get project information",
		Long: `Provide access to project information,
using: get -flags project [PROJECT_IDENTIFIERS]`,
		Run: project(&getProjectErr),
		PostRunE: func(command *cobra.Command, args []string) error {
			return getProjectErr
		},
	}
	getProjectCmd.PersistentFlags().Bool(consts.AllTenantsFlagName, false, "address all tenants of the user: --all-tenants")
	getProjectCmd.PersistentFlags().StringP(consts.TenantFlagName, string(consts.TenantFlagName[0]), "", "specify one of user tenant: -t first_tenant")

	return getProjectCmd
}

func project(outErr *error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		flags := cmd.Flags()

		serverFilter := model.ServerFilter{}
		serverFilter.AllTenants, *outErr = flags.GetBool(consts.AllTenantsFlagName)
		if *outErr != nil {
			return
		}
		serverFilter.TenantIdentifiers, *outErr = model.StringSetFromStringFlag(flags, consts.TenantFlagName)
		if *outErr != nil {
			return
		}
		// TODO validate serverFilter

		if len(args) == 0 {
			serverFilter.AllProjects = true
			serverFilter.ProjectIdentifiers = model.StringSet{}
		} else {
			set := model.StringSet{}
			for _, ti := range args {
				set.Put(ti)
			}
			serverFilter.ProjectIdentifiers = set
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

		_, projects, err := getProjectData(onlyCache, cache, serverFilter, permanentCacheFilePath)
		if err != nil {
			*outErr = err
			return
		}

		fmt.Printf("output flag= %s\n", output)
		fmt.Printf("projects: \n")
		for _, p := range projects {
			fmt.Printf("%s\n", p.Identifier)
		}
		fmt.Printf("========\n")
	}
}

func getProjectData(onlyCache bool, cache *model.ServerList, serverFilter model.ServerFilter,
	permanentCacheFilePath string) (map[iam.TenantUUID]iam.Tenant, map[iam.ProjectUUID]iam.Project, error) {
	var (
		tenants  map[iam.TenantUUID]iam.Tenant
		projects map[iam.ProjectUUID]iam.Project
	)

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

	} else {
		var err error
		tenants, err = vault.NewService().UpdateTenants(cache.Tenants, serverFilter.TenantIdentifiers)
		if err != nil {
			return nil, nil, err
		}
		projects, err = vault.NewService().UpdateProjects(cache.Projects, tenants, serverFilter.ProjectIdentifiers)
		if err != nil {
			return nil, nil, err
		}
		*cache = model.UpdateServerListCacheWithFreshValues(*cache, model.ServerList{
			Tenants:  tenants,
			Projects: projects,
		})

		model.SaveToFile(*cache, permanentCacheFilePath)
	}
	return tenants, projects, nil
}
