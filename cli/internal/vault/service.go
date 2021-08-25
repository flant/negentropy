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
	GetServersByFilter(model.ServerFilter) (*model.ServerList, error)
	SignPublicSSHCertificate(model.VaultSSHSignRequest) []byte
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
	if !filter.AllTenants && !filter.AllProjects {
		return v.getServersByTenantAndProject(filter)
	} else if !filter.AllTenants {
		return v.getServersByTenant(filter)
	} else if !filter.AllProjects {
		return v.getServersByProject(filter)
	} else {
		return v.getServers(filter)
	}
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
	jose.ParseSigned(token) // nolint:errCheck

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
	fmt.Printf("conn info: %#v", s.ConnectionInfo)
	return nil
}

func (v vaultService) getServersByTenantAndProject(filter model.ServerFilter) (*model.ServerList, error) {
	tenant, err := v.tenantByIdentifier(filter.TenantIdentifier)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenantAndProject, collecting tenant: %w", err)
	}
	tenants := map[iam.TenantUUID]iam.Tenant{tenant.UUID: *tenant}
	project, err := v.projectByTenantAndProjectIdentifier(tenant.UUID, filter.ProjectIdentifier)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenantAndProject, collecting project: %w", err)
	}
	projects := map[iam.ProjectUUID]iam.Project{project.UUID: *project}
	servers, err := v.vaultSession.GetServersByTenantAndProject(tenant.UUID, project.UUID,
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

func (v vaultService) getServersByTenant(filter model.ServerFilter) (*model.ServerList, error) {
	tenant, err := v.tenantByIdentifier(filter.TenantIdentifier)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenant, collecting tenant: %w", err)
	}
	tenants := map[iam.TenantUUID]iam.Tenant{tenant.UUID: *tenant}
	projects, err := v.projectsByTenant(tenant.UUID)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenant, collecting project: %w", err)
	}
	servers, err := v.vaultSession.GetServersByTenant(tenant.UUID, filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("getServersByTenant, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  tenants,
		Projects: projects,
		Servers:  servers,
	}, nil
}

func (v vaultService) getServersByProject(filter model.ServerFilter) (*model.ServerList, error) {
	allTenants, err := v.allTenants()
	if err != nil {
		return nil, fmt.Errorf("getServersByProject, collecting tenants: %w", err)
	}

	project, err := v.projectByTenantsAndProjectIdentifier(allTenants, filter.ProjectIdentifier)
	if err != nil {
		return nil, fmt.Errorf("getServersByProject, collecting project: %w", err)
	}
	servers, err := v.vaultSession.GetServersByTenantAndProject(project.TenantUUID, project.UUID,
		filter.ServerIdentifiers, filter.LabelSelector)
	if err != nil {
		return nil, fmt.Errorf("getServersByProject, collecting servers: %w", err)
	}
	return &model.ServerList{
		Tenants:  allTenants,
		Projects: map[iam.ProjectUUID]iam.Project{project.UUID: *project},
		Servers:  servers,
	}, nil
}

func (v vaultService) getServers(filter model.ServerFilter) (*model.ServerList, error) {
	allTenants, err := v.allTenants()
	if err != nil {
		return nil, fmt.Errorf("getServers, collecting tenants: %w", err)
	}

	allProjects, err := v.allProjectsByTenants(allTenants)
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

func (v vaultService) projectByTenantAndProjectIdentifier(tenantUUID iam.TenantUUID,
	projectIdentifier string) (*iam.Project, error) {
	projects, err := v.vaultSession.getProjects(tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("projectsByTenantsAndProjectIdentifiers: %w", err)
	}
	for i := range projects {
		project := projects[i]
		if project.Identifier == projectIdentifier {
			return &project, nil
		}
	}
	return nil, iam.ErrNotFound
}

func (v vaultService) projectsByTenant(tenantUUID iam.TenantUUID) (map[iam.ProjectUUID]iam.Project, error) {
	result := map[iam.ProjectUUID]iam.Project{}
	ps, err := v.vaultSession.getProjects(tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("projectsByTenant: %w", err)
	}
	for i := range ps {
		ps[i].TenantUUID = tenantUUID
		result[ps[i].UUID] = ps[i]
	}
	return result, nil
}

func (v vaultService) allTenants() (map[iam.TenantUUID]iam.Tenant, error) {
	result := map[iam.TenantUUID]iam.Tenant{}
	tanants, err := v.vaultSession.getTenants()
	if err != nil {
		return nil, fmt.Errorf("allTenants: %w", err)
	}
	for i := range tanants {
		result[tanants[i].UUID] = tanants[i]
	}
	return result, nil
}

func (v vaultService) tenantByIdentifier(tenantIdentifier string) (*iam.Tenant, error) {
	tenants, err := v.vaultSession.getTenants()
	if err != nil {
		return nil, fmt.Errorf("tenantByIdentifier:%w", err)
	}
	for i := range tenants {
		if tenantIdentifier != "" && tenants[i].Identifier == tenantIdentifier {
			return &tenants[i], nil
		}
	}
	return nil, nil
}

func (v vaultService) projectByTenantsAndProjectIdentifier(tenants map[iam.TenantUUID]iam.Tenant,
	projectIdentifier iam.ProjectUUID) (*iam.Project, error) {
	for tenantUUID := range tenants {
		ps, err := v.vaultSession.getProjects(tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("projectByTenantsAndProjectIdentifier: %w", err)
		}
		for i := range ps {
			if ps[i].Identifier == projectIdentifier {
				ps[i].TenantUUID = tenantUUID
				return &ps[i], nil
			}
		}
	}
	return nil, iam.ErrNotFound
}

func (v vaultService) allProjectsByTenants(tenants map[iam.TenantUUID]iam.Tenant) (map[iam.ProjectUUID]iam.Project, error) {
	result := map[iam.ProjectUUID]iam.Project{}
	for tenantUUID := range tenants {
		ps, err := v.vaultSession.getProjects(tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("allProjectsByTenants: %w", err)
		}
		for i := range ps {
			ps[i].TenantUUID = tenantUUID
			result[ps[i].UUID] = ps[i]
		}
	}
	return result, iam.ErrNotFound
}
