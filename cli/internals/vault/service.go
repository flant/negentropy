package vault

import (
	"encoding/json"
	"fmt"

	"gopkg.in/square/go-jose.v2"

	"github.com/flant/negentropy/cli/internals/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type VaultService interface {
	GetServerToken(ext.Server) (string, error)
	GetUser() iam.User
	GetServersByFilter(model.ServerFilter) (*model.ServerList, error)
	SignPublicSSHCertificate(model.VaultSSHSignRequest) []byte
}

type vaultService struct {
	vaultSession VaultSession
}

func (v vaultService) GetServerToken(server ext.Server) (string, error) {
	return v.vaultSession.GetServerToken(server)
}

func (v vaultService) GetUser() iam.User {
	return v.vaultSession.GetUser()
}

func (v vaultService) GetServersByFilter(filter model.ServerFilter) (*model.ServerList, error) {
	// sl.Tenant = vs.getTenantByIdentifier(filter.TenantIdentifier)
	// если в фильтре есть ограничения по проектам:
	//   projects := vs.getProjectsByTenant(&sl.Tenant)
	//   выгрести лишние проекты из ответа по фильтру
	//   для оставшихся проектов: LIST /tenant/<tenant.UUID>/project/<project.UUID>/query_server?<filter.RenderURIArgs()>
	// если ограничений нет, то
	//   LIST /tenant/<tenant.UUID>/query_server?<filter.RenderURIArgs()>

	// == имеем ServerList, осталось заполнить манифесты
	// для каждого сервера server.ConnectionInfo = vs.getServerManifest(server)
	// если есть bastion, то как-то его надо заинклудить
	// return serverList

	tenants, err := v.tenantsByIdentifiers(filter.TenantIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("GetServersByFilter:%w", err)
	}

	projects, err := v.projectsByTenantsAndProjectIdentifiers(tenants, filter.ProjectIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("GetServersByFilter:%w", err)
	}

	servers, err := v.serversByFilter(projects, filter.ServerIdentifiers, filter.LabelSelectors)
	return &model.ServerList{
		Tenants:  tenants,
		Projects: projects,
		Servers:  servers,
	}, nil
}

func (v vaultService) SignPublicSSHCertificate(req model.VaultSSHSignRequest) []byte {
	return v.vaultSession.SignPublicSSHCertificate(req)
}

func NewService() VaultService {
	return vaultService{NewVaultSession()}
}

func (v vaultService) tenantsByIdentifiers(tenantsIdentifiers []string) (map[iam.TenantUUID]iam.Tenant, error) {
	tenants, err := v.vaultSession.getTenants()
	if err != nil {
		return nil, fmt.Errorf("tenantsByIdentifiers:%w", err)
	}
	idSet := map[string]struct{}{}
	for i := range tenantsIdentifiers {
		idSet[tenantsIdentifiers[i]] = struct{}{}
	}
	result := map[iam.TenantUUID]iam.Tenant{}
	for i := range tenants {
		if _, ok := idSet[tenants[i].Identifier]; ok || len(idSet) == 0 {
			result[tenants[i].UUID] = tenants[i]
		}
	}
	return result, nil
}

func (v vaultService) projectsByTenantsAndProjectIdentifiers(tenants map[iam.TenantUUID]iam.Tenant,
	projectIdentifiers []iam.ProjectUUID) (map[iam.ProjectUUID]iam.Project, error) {
	projects := []iam.Project{}
	for tenantUUID := range tenants {
		ps, err := v.vaultSession.getProjects(tenantUUID)
		if err != nil {
			return nil, fmt.Errorf("projectsByTenantsAndProjectIdentifiers:%w", err)
		}
		for i := range ps {
			ps[i].TenantUUID = tenantUUID
		}

		projects = append(projects, ps...)
	}

	idSet := map[string]struct{}{}
	for i := range projectIdentifiers {
		idSet[projectIdentifiers[i]] = struct{}{}
	}
	result := map[iam.ProjectUUID]iam.Project{}
	for i := range projects {
		if _, ok := idSet[projects[i].Identifier]; ok || len(idSet) == 0 {
			result[projects[i].UUID] = projects[i]
		}
	}
	return result, nil
}

func (v vaultService) serversByFilter(projects map[iam.ProjectUUID]iam.Project, identifiers []string,
	selectors []string) ([]ext.Server, error) {

	var servers []ext.Server
	for projectUUID, project := range projects {
		ss, err := v.vaultSession.GetServers(project.TenantUUID, projectUUID)
		if err != nil {
			return nil, fmt.Errorf("serversByFilter:%w", err)
		}
		servers = append(servers, ss...)
	}
	idSet := map[string]struct{}{}
	for i := range identifiers {
		idSet[identifiers[i]] = struct{}{}
	}
	var result []ext.Server
	for i := range servers {
		if _, ok := idSet[servers[i].Identifier]; ok ||
			len(idSet) == 0 { // TODO add checking labels
			err := v.fillServerSecureData(&servers[i])
			if err != nil {
				return nil, fmt.Errorf("serversByFilter:%w", err)
			}

			result = append(result, servers[i])
		}
	}
	return result, nil
}

func (v vaultService) fillServerSecureData(s *ext.Server) error {
	token, err := v.vaultSession.GetServerToken(*s)
	if err != nil {
		return fmt.Errorf("FillServerSecureData:%w", err)
	}

	// TODO check signature
	jose.ParseSigned(token) // nolint:errCheck

	jwt, err := jose.ParseSigned(token)
	if err != nil {
		return err
	}

	payloadBytes := jwt.UnsafePayloadWithoutVerification()

	var server ext.Server

	err = json.Unmarshal(payloadBytes, &server)
	if err != nil {
		return err
	}
	s.ConnectionInfo = server.ConnectionInfo
	s.Fingerprint = server.Fingerprint
	return nil
}
