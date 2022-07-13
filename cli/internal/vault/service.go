package vault

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/cli/internal/model"
	"github.com/flant/negentropy/cli/pkg"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

type VaultService interface {
	// GetUser returns user which credentials are recognized by vault
	GetUser() (*auth.User, error)
	// UpdateServersByFilter returns ServerList synchronized with vault, according filter, using given ServerList as cache
	UpdateServersByFilter(model.ServerFilter, *model.ServerList) (*model.ServerList, error)
	SignPublicSSHCertificate(iam.TenantUUID, iam.ProjectUUID, []ext.ServerUUID, model.VaultSSHSignRequest) ([]byte, error)
	// UpdateTenants update oldTenants by vault requests, according specified identifiers given by args
	UpdateTenants(map[iam.TenantUUID]iam.Tenant, model.StringSet) (map[iam.TenantUUID]iam.Tenant, error)
	// UpdateProjects update oldProjects by vault requests, according specified identifiers given by args
	UpdateProjects(map[iam.ProjectUUID]iam.Project, map[iam.TenantUUID]iam.Tenant,
		model.StringSet) (map[iam.TenantUUID]iam.Tenant, map[iam.ProjectUUID]iam.Project, error)
}

type vaultService struct {
	cl pkg.VaultClient
}

func NewService() (VaultService, error) {
	defaultVaultClient, err := pkg.DefaultVaultClient()
	if err != nil {
		return nil, err
	}
	defaultService := &vaultService{
		cl: defaultVaultClient,
	}
	return defaultService, nil
}

func (v *vaultService) GetUser() (*auth.User, error) {
	return v.cl.GetUser()
}

// UpdateServersByFilter use oldServerList as cache and filter for returning synchronized with vault ServerList
func (v *vaultService) UpdateServersByFilter(filter model.ServerFilter, oldServerList *model.ServerList) (*model.ServerList, error) {
	var (
		newSl *model.ServerList
		err   error
	)
	if !filter.AllTenants && !filter.AllProjects {
		newSl, err = v.updateServerListByTenantAndProject(filter, oldServerList)
	} else if !filter.AllTenants {
		newSl, err = v.updateServerListByTenant(filter, oldServerList)
	} else if !filter.AllProjects {
		newSl, err = v.updateServerListByProject(filter, oldServerList)
	} else {
		newSl, err = v.updateServerList(filter, oldServerList)
	}
	if err != nil {
		return nil, fmt.Errorf("UpdateServersByFilter: %w", err)
	}
	return newSl, nil
}

func (v *vaultService) SignPublicSSHCertificate(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
	serverUUIDs []ext.ServerUUID, req model.VaultSSHSignRequest) ([]byte, error) {

	return v.cl.SignPublicSSHCertificate(tenantUUID, projectUUID, serverUUIDs, req)
}

func (v *vaultService) updateServerListByTenantAndProject(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	tenants, err := v.UpdateTenants(oldServerlist.Tenants, filter.TenantIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenantAndProject, collecting tenant: %w", err)
	}
	var tenantUUID iam.TenantUUID
	for tenantUUID = range tenants {
		break // the only one uuid should be presented
	}

	tenants, projects, err := v.UpdateProjects(oldServerlist.Projects, tenants, filter.ProjectIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenantAndProject, collecting project: %w", err)
	}
	var projectUUID iam.ProjectUUID
	for projectUUID = range projects {
		break // the only one uuid should be presented
	}
	servers, err := v.cl.GetServersByTenantAndProject(tenantUUID, projectUUID,
		filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenantAndProject, collecting servers: %w", err)
	}

	return &model.ServerList{
		Tenants:  tenants,
		Projects: projects,
		Servers:  serverMap(servers),
	}, nil
}

func serverMap(servers []ext.Server) map[ext.ServerUUID]ext.Server {
	result := map[ext.ServerUUID]ext.Server{}
	for _, s := range servers {
		result[s.UUID] = s
	}
	return result
}

func safeServerMap(servers []auth.SafeServer) map[ext.ServerUUID]auth.SafeServer {
	result := map[ext.ServerUUID]auth.SafeServer{}
	for _, s := range servers {
		result[s.UUID] = s
	}
	return result
}

// updateServerListByTenant update data using vault unsafe paths, so, returned serverList contains
// sensitive data only from given oldServerList
func (v *vaultService) updateServerListByTenant(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	tenants, err := v.UpdateTenants(oldServerlist.Tenants, filter.TenantIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenant, collecting tenant: %w", err)
	}
	var tenantUUID iam.TenantUUID
	for tenantUUID = range tenants {
		break // the only one uuid should be presented
	}

	tenants, projects, err := v.UpdateProjects(oldServerlist.Projects, tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenant, collecting project: %w", err)
	}
	servers, err := v.cl.GetSafeServersByTenant(tenantUUID, filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenant, collecting servers: %w", err)
	}
	serverMap, err := v.synchronizeSensitiveData(oldServerlist.Servers, safeServerMap(servers), filter)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenant, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  tenants,
		Projects: projects,
		Servers:  serverMap,
	}, nil
}

// updateServerListByProject update data using vault safe paths, so, returned serverList contains sensitive data
func (v *vaultService) updateServerListByProject(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	allTenants, err := v.UpdateTenants(oldServerlist.Tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByProject, collecting tenant: %w", err)
	}

	allTenants, projects, err := v.UpdateProjects(oldServerlist.Projects, allTenants, filter.ProjectIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByProject, collecting project: %w", err)
	}
	var project iam.Project
	for _, project = range projects {
		break // the only one project should be presented
	}

	servers, err := v.cl.GetServersByTenantAndProject(project.TenantUUID, project.UUID,
		filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByProject, collecting servers: %w", err)
	}

	return &model.ServerList{
		Tenants:  allTenants,
		Projects: projects,
		Servers:  serverMap(servers),
	}, nil
}

// updateServerList update data using vault unsafe paths, so, returned serverList contains
// sensitive data only from given oldServerList
func (v *vaultService) updateServerList(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	allTenants, err := v.UpdateTenants(oldServerlist.Tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("updateServerList, collecting tenant: %w", err)
	}
	allTenants, allProjects, err := v.UpdateProjects(oldServerlist.Projects, allTenants, nil)
	if err != nil {
		return nil, fmt.Errorf("updateServerList, collecting project: %w", err)
	}
	servers, err := v.cl.GetSafeServers(filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("updateServerList, collecting servers: %w", err)
	}
	serverMap, err := v.synchronizeSensitiveData(oldServerlist.Servers, safeServerMap(servers), filter)
	if err != nil {
		return nil, fmt.Errorf("updateServerList, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  allTenants,
		Projects: allProjects,
		Servers:  serverMap,
	}, nil
}

// synchronizeSensitiveData uses oldServers as cache return synchronized with vault slice of ext.Server
func (v *vaultService) synchronizeSensitiveData(oldServers map[ext.ServerUUID]ext.Server,
	newServers map[ext.ServerUUID]auth.SafeServer, filter model.ServerFilter) (map[ext.ServerUUID]ext.Server, error) {
	result := map[ext.ServerUUID]ext.Server{}
	for _, safeServer := range newServers {
		if oldS, ok := oldServers[safeServer.UUID]; !ok || oldS.Version != safeServer.Version ||
			oldS.ConnectionInfo.Hostname == "" || oldS.ConnectionInfo.Port == "" {
			servers, err := v.cl.GetServersByTenantAndProject(safeServer.TenantUUID, safeServer.ProjectUUID,
				filter.ServerIdentifiers, filter.LabelSelector)
			if err != nil {
				return nil, fmt.Errorf("synchronizeSensitiveData, collecting servers with sensitive data: %w", err)
			}
			for _, s := range servers {
				oldServers[s.UUID] = s
			}
			result[safeServer.UUID] = oldServers[safeServer.UUID]
		} else {
			result[safeServer.UUID] = oldS
		}
	}
	return result, nil
}

// UpdateTenants return user tenants synchronized with vault
func (v *vaultService) UpdateTenants(oldTenants map[iam.TenantUUID]iam.Tenant,
	tenantIdentifiers model.StringSet) (map[iam.TenantUUID]iam.Tenant, error) {
	result := map[iam.TenantUUID]iam.Tenant{}
	// enough default client
	tenants, err := v.cl.GetTenants()
	if err != nil {
		return nil, fmt.Errorf("UpdateTenants: %w", err)
	}
	for _, t := range tenants {
		var tenant iam.Tenant

		if oldTenant, ok := oldTenants[t.UUID]; !ok ||
			oldTenant.Version != t.Version {
			tmp, err := v.cl.GetTenantByUUID(t.UUID)
			if errors.Is(err, consts.ErrAccessForbidden) {
				// this tenant is prohibited, doesn't add it to list
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("UpdateTenants: %w", err)
			}
			tenant = *tmp
		} else {
			tenant = oldTenant
		}
		if tenantIdentifiers.IsEmpty() || tenantIdentifiers.Contains(tenant.Identifier) {
			result[tenant.UUID] = tenant
		}
	}
	return result, nil
}

// UpdateProjects return user projects and tenants synchronized with vault
func (v *vaultService) UpdateProjects(oldProjects map[iam.ProjectUUID]iam.Project, tenants map[iam.TenantUUID]iam.Tenant,
	projectIdentifiers model.StringSet) (map[iam.TenantUUID]iam.Tenant, map[iam.ProjectUUID]iam.Project, error) {
	resultTenants := map[iam.TenantUUID]iam.Tenant{}
	result := map[iam.ProjectUUID]iam.Project{}
	var projects []auth.Project
	for tenantUUID, tenant := range tenants {
		ps, err := v.cl.GetProjects(tenantUUID)
		if errors.Is(err, consts.ErrAccessForbidden) {
			// this tenant is prohibited, doesn't add it to list
			continue
		}
		resultTenants[tenantUUID] = tenant
		if err != nil {
			return nil, nil, fmt.Errorf("UpdateProjects: %w", err)
		}
		projects = append(projects, ps...)
	}

	for _, p := range projects {
		var project iam.Project

		if oldProject, ok := oldProjects[p.UUID]; !ok ||
			oldProject.Version != p.Version {
			tmp, err := v.cl.GetProjectByUUID(p.TenantUUID, p.UUID)
			if err != nil {
				return nil, nil, fmt.Errorf("UpdateProjects: %w", err)
			}
			project = *tmp
		} else {
			project = oldProject
		}
		if projectIdentifiers.IsEmpty() || projectIdentifiers.Contains(project.Identifier) {
			result[project.UUID] = project
		}
	}
	return resultTenants, result, nil
}
