package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"main/pkg/iam"
	"os"
	"strings"

	vault_api "github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/authd"
	v1 "github.com/flant/negentropy/authd/pkg/api/v1"
)

type VaultSession struct {
	Client *vault_api.Client
}

type ServerFilter struct {
	// некое описание labelSelector-ов
	TenantIdentifier   string
	ProjectIdentifiers []string
	ServerIdentifiers  []string
	// LabelSelector
}

type VaultSSHSignRequest struct {
	PublicKey       string `json:"public_key"`
	ValidPrincipals string `json:"valid_principals"`
}

type VaultTenantsResponse struct {
	Data struct {
		Tenants []iam.Tenant `json:"tenants"`
	} `json:"data"`
}

type VaultProjectsResponse struct {
	Data struct {
		Projects []iam.Project `json:"projects"`
	} `json:"data"`
}

type VaultServersResponse struct {
	Data struct {
		Servers []iam.Server `json:"servers"`
	} `json:"data"`
}

type VaultServerTokenResponse struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

func (sf *ServerFilter) RenderURIArgs() string {
	// ?name[]=db1&name[]=db-2&...
	// labelselector=...
	return ""
}

func (vs *VaultSession) Init() {
	authdClient := authd.NewAuthdClient("/run/authd.sock")

	req := v1.NewLoginRequest().
		WithRoles(v1.NewRoleWithClaim("*", map[string]string{})).
		WithServerType(v1.AuthServer)

	err := authdClient.OpenVaultSession(req)
	if err != nil {
		panic(err)
	}

	vaultClient, err := authdClient.NewVaultClient()
	if err != nil {
		panic(err)
	}

	vs.Client = vaultClient
}

func (vs *VaultSession) Request(method, requestPath string) ([]byte, error) {
	req := vs.Client.NewRequest(method, requestPath)
	resp, err := vs.Client.RawRequest(req)
	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}

func (vs *VaultSession) RequestTenants() ([]iam.Tenant, error) {
	vaultTenantsResponseBytes, err := vs.Request("LIST", "/v1/auth/flant_iam_auth/tenant")
	if err != nil {
		return []iam.Tenant{}, err
	}

	var vaultTenantsResponse VaultTenantsResponse
	err = json.Unmarshal(vaultTenantsResponseBytes, &vaultTenantsResponse)
	if err != nil {
		return []iam.Tenant{}, err
	}

	return vaultTenantsResponse.Data.Tenants, nil
}

func (vs *VaultSession) RequestProjects(tenant *iam.Tenant) ([]iam.Project, error) {
	vaultProjectsResponseBytes, err := vs.Request("LIST", fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project", tenant.UUID))
	if err != nil {
		return []iam.Project{}, err
	}

	var vaultProjectsResponse VaultProjectsResponse
	err = json.Unmarshal(vaultProjectsResponseBytes, &vaultProjectsResponse)
	if err != nil {
		return []iam.Project{}, err
	}

	projects := vaultProjectsResponse.Data.Projects
	for i := range projects {
		projects[i].Tenant = tenant
	}
	return projects, nil
}

// TODO filter
// TODO warnings
func (vs *VaultSession) RequestServers(tenant *iam.Tenant, project *iam.Project) ([]iam.Server, error) {
	var vaultServersResponse VaultServersResponse
	var requestPath string
	if project != nil {
		requestPath = fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project/%s/query_server", tenant.UUID, project.UUID)
	} else {
		requestPath = fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/query_server", tenant.UUID)
	}

	vaultServersResponseBytes, err := vs.Request("GET", requestPath)
	if err != nil {
		return []iam.Server{}, err
	}

	err = json.Unmarshal(vaultServersResponseBytes, &vaultServersResponse)
	if err != nil {
		return []iam.Server{}, err
	}

	servers := vaultServersResponse.Data.Servers

	//servers = []iam.Server{
	//	iam.Server{
	//		UUID:        "aaa",
	//		Identifier:  "vfe1",
	//		ProjectUUID: "0831b50a-dcfc-406c-b7bf-8d52619e8de5",
	//	},
	//}
	if project != nil {
		for i := range servers {
			servers[i].Project = project
		}
	}

	return servers, nil
}

func (vs *VaultSession) RequestServerToken(server *iam.Server) (string, error) {
	vaultServerTokenResponseBytes, err := vs.Request("GET", fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project/%s/server/%s", server.Project.Tenant.UUID, server.Project.UUID, server.UUID))
	if err != nil {
		return "", err
	}

	var vaultServerTokenResponse VaultServerTokenResponse
	err = json.Unmarshal(vaultServerTokenResponseBytes, &vaultServerTokenResponse)
	if err != nil {
		return "", err
	}

	return vaultServerTokenResponse.Data.Token, nil
}

func (vs *VaultSession) GetSSHUser() iam.User {
	// достать   из vault инфу про текущего юзера
	userUUID := os.Getenv("USER_UUID")
	if userUUID == "" {
		userUUID = "uuu"
	}
	userFullID := os.Getenv("USER_FULL_ID")
	a := strings.Split(userFullID, "@")

	if userFullID == "" {
		userFullID = "fluser"
	}
	fmt.Printf("user %s, identifier %s\n", userUUID, userFullID)
	return iam.User{UUID: userUUID, FullIdentifier: a[0]}
}

func (vs *VaultSession) getTenantByIdentifier(identifier string) (iam.Tenant, error) {
	// LIST /tenant -> ищем наш
	//tenant := iam.Tenant{
	//	Identifier: identifier,
	//	UUID: <uuid>
	//}
	// return tenant

	// TODO cache

	tenants, err := vs.RequestTenants()
	if err != nil {
		return iam.Tenant{}, err
	}

	for _, tenant := range tenants {
		if tenant.Identifier == identifier {
			return tenant, nil
		}
	}

	return iam.Tenant{}, nil
}

func (vs *VaultSession) QueryServer(filter ServerFilter) (iam.ServerList, error) {
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

	tenant, err := vs.getTenantByIdentifier(filter.TenantIdentifier)
	if err != nil {
		return iam.ServerList{}, err
	}

	sl := iam.ServerList{
		Tenant:   tenant,
		Projects: []iam.Project{},
		Servers:  []iam.Server{},
	}
	sl.Projects, err = vs.RequestProjects(&sl.Tenant)
	if err != nil {
		return iam.ServerList{}, err
	}

	if len(filter.ProjectIdentifiers) == 0 {
		sl.Servers, err = vs.RequestServers(&sl.Tenant, nil)
		if err != nil {
			return iam.ServerList{}, err
		}

		for i, server := range sl.Servers {
			for j, project := range sl.Projects {
				if server.ProjectUUID == project.UUID {
					sl.Servers[i].Project = &sl.Projects[j]
				}
			}
		}
	} else {
		for _, projectIdentifier := range filter.ProjectIdentifiers {
			for i, project := range sl.Projects {
				if project.Identifier == projectIdentifier {
					servers, err := vs.RequestServers(&sl.Tenant, &sl.Projects[i])
					if err != nil {
						return iam.ServerList{}, err
					}
					sl.Servers = append(sl.Servers, servers...)
				}
			}
		}
	}

	//sl := iam.ServerList{
	//	Tenant: iam.Tenant{
	//		Identifier: "aaa",
	//		UUID:       "qweqwe",
	//	},
	//	Projects: []iam.Project{},
	//}
	//
	//sl.Projects = append(sl.Projects, iam.Project{UUID: "bbb", Identifier: "myproject", Tenant: &sl.Tenant})
	//sl.Servers = append(sl.Servers, iam.Server{Identifier: "node-1", UUID: "ccc", Project: &sl.Projects[0], ConnectionInfo: iam.ServerConnectionInfo{Hostname: "95.216.34.23", Port: 2202}})
	return sl, nil
}

func (vs *VaultSession) SignPublicSSHCertificate(vaultReq map[string]interface{}) []byte {
	//tenant, _ := vs.getTenantByIdentifier("1tv")
	//fmt.Println(tenant)
	//
	//fmt.Println(vs.getProjectsByTenant(&tenant))

	ssh := vs.Client.SSHWithMountPoint("ssh")
	secret, err := ssh.SignKey("signer", vaultReq)
	if err != nil {
		panic(err)
	}

	return []byte(secret.Data["signed_key"].(string))
}
