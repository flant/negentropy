package vault

import (
	"encoding/json"
	"fmt"

	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/cli/internal/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extension_server_access/model"
)

type VaultService interface {
	GetServerToken(ext.Server) (string, error)
	GetUser() (*auth.User, error)
	UpdateServersByFilter(model.ServerFilter, *model.ServerList) (*model.ServerList, error)
	SignPublicSSHCertificate(model.VaultSSHSignRequest) ([]byte, error)
	FillServerSecureData(s *ext.Server) error
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
	fmt.Printf("\nfilter:\n %#v\n", filter)
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
	for i := range newSl.Servers {
		// TODO вот это должно рабоать по протуханию JWT или по отсутствию СonnectionInfo
		err := v.FillServerSecureData(&newSl.Servers[i])
		if err != nil {
			fmt.Printf("Error getting server secure data: %s", err)
		}
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
	s.ConnectionInfo = server.ConnectionInfo
	s.Fingerprint = server.Fingerprint
	return nil
}

func (v vaultService) updateServerListByTenantAndProject(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	tenants, err := v.updateTenants(oldServerlist.Tenants, &filter.TenantIdentifier)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenantAndProject, collecting project: %w", err)
	}
	var tenantUUID iam.TenantUUID
	for tenantUUID = range tenants {
		break // the only one uuid should be presented
	}

	projects, err := v.updateProjects(oldServerlist.Projects, tenants, &filter.ProjectIdentifier)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenantAndProject, collecting project: %w", err)
	}
	var projectUUID iam.ProjectUUID
	for projectUUID = range projects {
		break // the only one uuid should be presented
	}

	servers, err := v.vaultSession.GetServersByTenantAndProject(tenantUUID, projectUUID,
		filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenantAndProject, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  tenants,
		Projects: projects,
		Servers:  servers,
	}, nil
}

// updateTenants return user tenants
// if tenantIdentifier not nil, only one tenant can be returned
func (v vaultService) updateTenants(oldTenants map[iam.TenantUUID]iam.Tenant, tenantIdentifier *string) (map[iam.TenantUUID]iam.Tenant, error) {
	result := map[iam.TenantUUID]iam.Tenant{}
	tenants, err := v.vaultSession.getTenants()
	if err != nil {
		return nil, fmt.Errorf("updateTenants: %w", err)
	}
	for _, t := range tenants {
		var tenant iam.Tenant

		if oldTenant, ok := oldTenants[t.UUID]; !ok ||
			oldTenant.Version != t.Version {
			tmp, err := v.vaultSession.getTenantByUUID(t.UUID)
			if err != nil {
				return nil, fmt.Errorf("updateTenants: %w", err)
			}
			tenant = *tmp
		} else {
			tenant = oldTenant
		}
		if tenantIdentifier == nil || tenant.Identifier == *tenantIdentifier {
			result[tenant.UUID] = tenant
		}
	}
	return result, nil
}

// updateTenants return user tenants
// if tenantIdentifier not nil, only one tenant can be returned
func (v vaultService) updateProjects(oldProjects map[iam.ProjectUUID]iam.Project, tenants map[iam.TenantUUID]iam.Tenant,
	projectIdentifier *string) (map[iam.ProjectUUID]iam.Project, error) {
	result := map[iam.ProjectUUID]iam.Project{}
	var projects []auth.SafeProject
	for tenantUUID := range tenants {
		ps, err := v.vaultSession.getProjects(tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("updateTenants: %w", err)
		}
		projects = append(projects, ps...)
	}

	for _, p := range projects {
		var project iam.Project

		if oldProject, ok := oldProjects[p.UUID]; !ok ||
			oldProject.Version != p.Version {
			tmp, err := v.vaultSession.getProjectByUUIDs(p.TenantUUID, p.UUID)
			if err != nil {
				return nil, fmt.Errorf("updateTenants: %w", err)
			}
			project = *tmp
		} else {
			project = oldProject
		}
		if projectIdentifier == nil || project.Identifier == *projectIdentifier {
			result[project.UUID] = project
		}
	}
	return result, nil
}

func (v vaultService) updateServerListByTenant(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	tenants, err := v.updateTenants(oldServerlist.Tenants, &filter.TenantIdentifier)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenant, collecting project: %w", err)
	}
	var tenantUUID iam.TenantUUID
	for tenantUUID = range tenants {
		break // the only one uuid should be presented
	}

	projects, err := v.updateProjects(oldServerlist.Projects, tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenant, collecting project: %w", err)
	}

	servers, err := v.vaultSession.GetServersByTenant(tenantUUID, filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenant, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  tenants,
		Projects: projects,
		Servers:  servers,
	}, nil
}

func (v vaultService) updateServerListByProject(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	allTenants, err := v.updateTenants(oldServerlist.Tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("getServersByProject, collecting project: %w", err)
	}

	projects, err := v.updateProjects(oldServerlist.Projects, allTenants, &filter.ProjectIdentifier)
	if err != nil {
		return nil, fmt.Errorf("getServersByProject, collecting project: %w", err)
	}
	var project iam.Project
	for _, project = range projects {
		break // the only one project should be presented
	}

	servers, err := v.vaultSession.GetServersByTenantAndProject(project.TenantUUID, project.UUID,
		filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("getServersByProject, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  allTenants,
		Projects: projects,
		Servers:  servers,
	}, nil
}

func (v vaultService) updateServerList(filter model.ServerFilter, oldServerlist *model.ServerList) (*model.ServerList, error) {
	allTenants, err := v.updateTenants(oldServerlist.Tenants, nil)
	if err != nil {
		return nil, fmt.Errorf("getServers, collecting project: %w", err)
	}

	allProjects, err := v.updateProjects(oldServerlist.Projects, allTenants, nil)
	if err != nil {
		return nil, fmt.Errorf("getServers, collecting project: %w", err)
	}
	servers, err := v.vaultSession.GetServers(filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("getServers, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  allTenants,
		Projects: allProjects,
		Servers:  servers,
	}, nil
}
