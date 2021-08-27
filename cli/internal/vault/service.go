package vault

import (
	"encoding/json"
	"fmt"

	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/cli/internal/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
)

type VaultService interface {
	GetServerToken(ext.Server) (string, error)
	GetUser() (*auth.User, error)
	UpdateServersByFilter(model.ServerFilter, *model.ServerList) (*model.ServerList, error)
	SignPublicSSHCertificate(model.VaultSSHSignRequest) ([]byte, error)
	FillServerSecureData(s *ext.Server) error
	// UpdateTenants update oldTenants by vault requests, according specified identifiers given by args
	UpdateTenants(map[iam.TenantUUID]iam.Tenant, model.StringSet) (map[iam.TenantUUID]iam.Tenant, error)
	// UpdateProjects update oldProjects by vault requests, according specified identifiers given by args
	UpdateProjects(map[iam.ProjectUUID]iam.Project, map[iam.TenantUUID]iam.Tenant,
		model.StringSet) (map[iam.ProjectUUID]iam.Project, error)
}

type vaultService struct {
	vaultSession VaultSession
}

func (v vaultService) GetServerToken(server ext.Server) (string, error) {
	return v.vaultSession.GetServerToken(server)
}

func (v vaultService) GetUser() (*auth.User, error) {
	return v.vaultSession.GetUser()
}

func (v vaultService) UpdateServersByFilter(filter model.ServerFilter, serverList *model.ServerList) (*model.ServerList, error) {
	var (
		newSl *model.ServerList
		err   error
	)
	if !filter.AllTenants && !filter.AllProjects {
		newSl, err = v.updateServerListByTenantAndProject(filter, serverList)
	} else if !filter.AllTenants {
		newSl, err = v.updateServerListByTenant(filter, serverList)
	} else if !filter.AllProjects {
		newSl, err = v.updateServerListByProject(filter, serverList)
	} else {
		newSl, err = v.updateServerList(filter, serverList)
	}
	if err != nil {
		return nil, fmt.Errorf("UpdateServersByFilter: %w", err)
	}
	for serverUUID, s := range newSl.Servers {
		// TODO вот это должно рабоать по протуханию JWT или по отсутствию СonnectionInfo
		err := v.FillServerSecureData(&s)
		if err != nil {
			return nil, fmt.Errorf("getting server secure data: %w", err)
		}
		newSl.Servers[serverUUID] = s
	}
	return newSl, nil
}

func (v vaultService) SignPublicSSHCertificate(req model.VaultSSHSignRequest) ([]byte, error) {
	return v.vaultSession.SignPublicSSHCertificate(req)
}

func NewService() VaultService {
	return vaultService{NewVaultSession()}
}

func (v vaultService) FillServerSecureData(s *ext.Server) error {
	token, err := v.vaultSession.GetServerToken(*s)
	if err != nil {
		return fmt.Errorf("FillServerSecureData, vault requesting: %w", err)
	}

	// TODO check signature
	jwt, err := jose.ParseSigned(token)
	if err != nil {
		return fmt.Errorf("FillServerSecureData, parsing jwt: %w", err)
	}

	payloadBytes := jwt.UnsafePayloadWithoutVerification()

	var server ext.Server

	err = json.Unmarshal(payloadBytes, &server)
	if err != nil {
		return fmt.Errorf("FillServerSecureData, unmarshalling jwt: %w", err)
	}
	s.Identifier = server.Identifier
	s.ConnectionInfo = server.ConnectionInfo
	s.Fingerprint = server.Fingerprint
	return nil
}

func (v vaultService) updateServerListByTenantAndProject(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	tenants, err := v.UpdateTenants(oldServerlist.Tenants, filter.TenantIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenantAndProject, collecting tenant: %w", err)
	}
	var tenantUUID iam.TenantUUID
	for tenantUUID = range tenants {
		break // the only one uuid should be presented
	}

	projects, err := v.UpdateProjects(oldServerlist.Projects, tenants, filter.ProjectIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenantAndProject, collecting project: %w", err)
	}
	var projectUUID iam.ProjectUUID
	for projectUUID = range projects {
		break // the only one uuid should be presented
	}

	servers, err := v.vaultSession.GetServersByTenantAndProject(tenantUUID, projectUUID,
		filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenantAndProject, collecting servers: %w", err)
	}
	serverMap, err := v.updateServers(oldServerlist.Servers, servers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenantAndProject, collecting servers: %w", err)
	}

	return &model.ServerList{
		Tenants:  tenants,
		Projects: projects,
		Servers:  serverMap,
	}, nil
}

// UpdateTenants return user tenants
// if tenantIdentifier not nil, only one tenant can be returned
func (v vaultService) UpdateTenants(oldTenants map[iam.TenantUUID]iam.Tenant,
	tenantIdentifiers model.StringSet) (map[iam.TenantUUID]iam.Tenant, error) {
	result := map[iam.TenantUUID]iam.Tenant{}
	tenants, err := v.vaultSession.getTenants()
	if err != nil {
		return nil, fmt.Errorf("UpdateTenants: %w", err)
	}
	for _, t := range tenants {
		var tenant iam.Tenant

		if oldTenant, ok := oldTenants[t.UUID]; !ok ||
			oldTenant.Version != t.Version {
			tmp, err := v.vaultSession.getTenantByUUID(t.UUID)
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

// UpdateTenants return user tenants
// if tenantIdentifier not nil, only one tenant can be returned
func (v vaultService) UpdateProjects(oldProjects map[iam.ProjectUUID]iam.Project, tenants map[iam.TenantUUID]iam.Tenant,
	projectIdentifiers model.StringSet) (map[iam.ProjectUUID]iam.Project, error) {
	result := map[iam.ProjectUUID]iam.Project{}
	var projects []auth.SafeProject
	for tenantUUID := range tenants {
		ps, err := v.vaultSession.getProjects(tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("UpdateProjects: %w", err)
		}
		projects = append(projects, ps...)
	}

	for _, p := range projects {
		var project iam.Project

		if oldProject, ok := oldProjects[p.UUID]; !ok ||
			oldProject.Version != p.Version {
			tmp, err := v.vaultSession.getProjectByUUIDs(p.TenantUUID, p.UUID)
			if err != nil {
				return nil, fmt.Errorf("UpdateProjects: %w", err)
			}
			project = *tmp
		} else {
			project = oldProject
		}
		if projectIdentifiers.IsEmpty() || projectIdentifiers.Contains(project.Identifier) {
			result[project.UUID] = project
		}
	}
	return result, nil
}

func (v vaultService) updateServerListByTenant(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	tenants, err := v.UpdateTenants(oldServerlist.Tenants, filter.TenantIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenant, collecting tenant: %w", err)
	}
	var tenantUUID iam.TenantUUID
	for tenantUUID = range tenants {
		break // the only one uuid should be presented
	}

	projects, err := v.UpdateProjects(oldServerlist.Projects, tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenant, collecting project: %w", err)
	}

	servers, err := v.vaultSession.GetServersByTenant(tenantUUID, filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenant, collecting servers: %w", err)
	}
	serverMap, err := v.updateServers(oldServerlist.Servers, servers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByTenant, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  tenants,
		Projects: projects,
		Servers:  serverMap,
	}, nil
}

func (v vaultService) updateServerListByProject(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	allTenants, err := v.UpdateTenants(oldServerlist.Tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByProject, collecting tenant: %w", err)
	}

	projects, err := v.UpdateProjects(oldServerlist.Projects, allTenants, filter.ProjectIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByProject, collecting project: %w", err)
	}
	var project iam.Project
	for _, project = range projects {
		break // the only one project should be presented
	}

	servers, err := v.vaultSession.GetServersByTenantAndProject(project.TenantUUID, project.UUID,
		filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByProject, collecting servers: %w", err)
	}
	serverMap, err := v.updateServers(oldServerlist.Servers, servers)
	if err != nil {
		return nil, fmt.Errorf("updateServerListByProject, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  allTenants,
		Projects: projects,
		Servers:  serverMap,
	}, nil
}

func (v vaultService) updateServerList(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	allTenants, err := v.UpdateTenants(oldServerlist.Tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("updateServerList, collecting tenant: %w", err)
	}

	allProjects, err := v.UpdateProjects(oldServerlist.Projects, allTenants, nil)
	if err != nil {
		return nil, fmt.Errorf("updateServerList, collecting project: %w", err)
	}
	servers, err := v.vaultSession.GetServers(filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("updateServerList, collecting servers: %w", err)
	}
	serverMap, err := v.updateServers(oldServerlist.Servers, servers)
	if err != nil {
		return nil, fmt.Errorf("updateServerList, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  allTenants,
		Projects: allProjects,
		Servers:  serverMap,
	}, nil
}

func (v vaultService) updateServers(oldServers map[ext.ServerUUID]ext.Server,
	newServers []ext.Server) (map[ext.ServerUUID]ext.Server, error) {
	result := map[ext.ServerUUID]ext.Server{}
	for _, s := range newServers {
		if oldS, ok := oldServers[s.UUID]; !ok || oldS.Version != s.Version {
			err := v.FillServerSecureData(&s)
			if err != nil {
				return nil, fmt.Errorf("updateServers: %w", err)
			}
		} else {
			s = oldS
		}
		result[s.UUID] = s
	}
	return result, nil
}
