package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/flant/negentropy/authd"

	authdapi "github.com/flant/negentropy/authd/pkg/api/v1"

	vault_api "github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/cli/internal/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	auth_ext "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

// VaultClient wrap requests to vault
// knows about needed roles, can escalate roles, if run by DefaultVaultClient
type VaultClient interface {
	GetTenants() ([]iam.Tenant, error)
	GetProjects(tenantUUID iam.TenantUUID) ([]auth.Project, error)
	GetUser() (*auth.User, error)
	GetServersByTenantAndProject(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
		serverIdentifiers []string, labelSelector string) ([]ext.Server, error)
	GetSafeServersByTenant(tenantUUID iam.TenantUUID, serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error)
	GetSafeServers(serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error)
	SignPublicSSHCertificate(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
		serverUUIDs []ext.ServerUUID, vaultReq model.VaultSSHSignRequest) ([]byte, error)
	GetTenantByUUID(tenantUUID string) (*iam.Tenant, error)
	GetProjectByUUID(tenantUUID string, projectUUID string) (*iam.Project, error)
	GetTenantByIdentifier(tenantIdentifier string) (*iam.Tenant, error)
	GetProjectByIdentifier(tenantUUID iam.TenantUUID, projectIdentifier string) (*auth.Project, error)
	RegisterServer(server ext.Server) (ext.ServerUUID, iam.MultipassJWT, error)
	UpdateServerConnectionInfo(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
		serverUUID ext.ServerUUID, connInfo ext.ConnectionInfo) (*ext.Server, error)
}

type vaultClient struct {
	*vault_api.Client  // authorized client
	roles              []authdapi.RoleWithClaim
	allowEscalateRoles bool
}

func ConfiguredVaultClient(authorizedClient *vault_api.Client, roles []authdapi.RoleWithClaim) VaultClient {
	return &vaultClient{
		Client:             authorizedClient,
		roles:              roles,
		allowEscalateRoles: false,
	}
}

func DefaultVaultClient() (VaultClient, error) {
	defaultClient := &vaultClient{allowEscalateRoles: true}
	err := defaultClient.checkForRolesAndUpdateClient()
	return defaultClient, err
}

func VaultClientAuthorizedWithSAPass(vaultURL string, password iam.ServiceAccountPassword,
	roles []authdapi.RoleWithClaim) (VaultClient, error) {
	cl, err := vault_api.NewClient(vault_api.DefaultConfig())
	if err != nil {
		return nil, err
	}
	err = cl.SetAddress(vaultURL)
	if err != nil {
		return nil, err
	}

	secret, err := cl.Logical().Write("/auth/flant_iam_auth/login", map[string]interface{}{
		"method":                          "sapassword",
		"service_account_password_uuid":   password.UUID,
		"service_account_password_secret": password.Secret,
		"roles":                           roles,
	})
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Auth == nil {
		return nil, fmt.Errorf("expect not nil secret.Auth, got secret:%#v", secret)
	}
	cl.SetToken(secret.Auth.ClientToken)
	return ConfiguredVaultClient(cl, roles), nil
}

func (vc *vaultClient) checkForRolesAndUpdateClient(neededRoles ...authdapi.RoleWithClaim) error {
	var extraRoles []authdapi.RoleWithClaim = nil
	for _, role := range neededRoles {
		exists := false
		for _, existedRole := range vc.roles {
			if existedRole.Role == role.Role &&
				existedRole.TenantUUID == role.TenantUUID &&
				existedRole.ProjectUUID == role.ProjectUUID {
				exists = true
				break
			}
		}
		if !exists {
			extraRoles = append(extraRoles, role)
		}
	}
	if len(extraRoles) == 0 && vc.Client != nil {
		return nil // no needs to relogin
	}
	if len(extraRoles) == 0 {
		extraRoles = []authdapi.RoleWithClaim{{
			Role: TenantsListRole,
		}}
	}
	if !vc.allowEscalateRoles {
		return fmt.Errorf("vault client run in 'allowEscalateRoles=false' mode, needs extra roles: %#v", extraRoles)
	}
	roles := append(vc.roles, extraRoles...)
	var authdSocketPath string
	if authdSocketPath = os.Getenv("AUTHD_SOCKET_PATH"); authdSocketPath == "" {
		authdSocketPath = "/run/authd.sock"
	}
	authdClient := authd.NewAuthdClient(authdSocketPath)

	req := authdapi.NewLoginRequest().
		WithRoles(roles...).
		WithServerType(authdapi.AuthServer)

	err := authdClient.OpenVaultSession(req)
	if err != nil {
		return fmt.Errorf("checkForRolesAndUpdateClient: %w", err)
	}

	vaultClient, err := authdClient.NewVaultClient()
	if err != nil {
		return fmt.Errorf("checkForRolesAndUpdateClient: %w", err)
	}
	vc.roles = roles
	vc.Client = vaultClient
	return nil
}

func (vc vaultClient) makeRequest(method, requestPath string, params url.Values, bodyBytes []byte) ([]byte, error) {
	req := vc.NewRequest(method, requestPath)
	if params != nil {
		req.Params = params
	}
	if len(bodyBytes) > 0 {
		req.BodyBytes = bodyBytes
	}
	resp, err := vc.Client.RawRequest(req)
	defer resp.Body.Close() // nolint:errcheck
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (vc *vaultClient) GetTenants() ([]iam.Tenant, error) {
	// allowed by default
	vaultTenantsResponseBytes, err := vc.makeRequest("LIST", "/v1/auth/flant_iam_auth/tenant", nil, nil)
	if err != nil {
		return nil, err
	}
	var vaultTenantsResponse struct {
		Data struct {
			Tenants []iam.Tenant `json:"tenants"`
		} `json:"data"`
	}
	err = json.Unmarshal(vaultTenantsResponseBytes, &vaultTenantsResponse)
	if err != nil {
		return nil, err
	}
	return vaultTenantsResponse.Data.Tenants, nil
}

func (vc *vaultClient) GetProjects(tenantUUID iam.TenantUUID) ([]auth.Project, error) {
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role:       TenantReadAuthRole,
		TenantUUID: tenantUUID,
	})
	if err != nil {
		return nil, err
	}
	vaultProjectsResponseBytes, err := vc.makeRequest("LIST",
		fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project", tenantUUID), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("getProjects: %w", err)
	}
	var vaultProjectsResponse struct {
		Data struct {
			Projects []auth.Project `json:"projects"`
		} `json:"data"`
	}
	err = json.Unmarshal(vaultProjectsResponseBytes, &vaultProjectsResponse)
	if err != nil {
		return nil, fmt.Errorf("getProjects: %w", err)
	}
	projects := vaultProjectsResponse.Data.Projects
	return projects, nil
}

// GetServersByTenantAndProject returns server full info
func (vc *vaultClient) GetServersByTenantAndProject(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
	serverIdentifiers []string, labelSelector string) ([]ext.Server, error) {
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role:        ServersQueryRole,
		TenantUUID:  tenantUUID,
		ProjectUUID: projectUUID,
	})
	if err != nil {
		return nil, err
	}
	requestPath := fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project/%s/query_server", tenantUUID, projectUUID)

	servers, err := vc.getServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.GetServersByTenantAndProject: %w", err)
	}
	return servers, nil
}

func (vc *vaultClient) GetUser() (*auth.User, error) {
	// allowed by default
	vaultResponseBytes, err := vc.makeRequest("GET", "/v1/auth/flant_iam_auth/vst_owner", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get_user:%w", err)
	}
	var vaultUserMultipasOwnerResponse struct {
		Data struct {
			User auth.User `json:"user"`
		} `json:"data"`
	}

	err = json.Unmarshal(vaultResponseBytes, &vaultUserMultipasOwnerResponse)
	if err != nil {
		return nil, fmt.Errorf("get_user:%w", err)
	}
	return &vaultUserMultipasOwnerResponse.Data.User, nil
}

func (vc *vaultClient) SignPublicSSHCertificate(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
	serverUUIDs []ext.ServerUUID, vaultReq model.VaultSSHSignRequest) ([]byte, error) {
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role:        SSHOpenRole,
		TenantUUID:  tenantUUID,
		ProjectUUID: projectUUID,
		Claim: map[string]interface{}{"ttl": "720m",
			"max_ttl": "1440m",
			"servers": serverUUIDs},
	})
	if err != nil {
		return nil, err
	}

	var reqMap map[string]interface{}
	data, _ := json.Marshal(vaultReq)
	err = json.Unmarshal(data, &reqMap)
	if err != nil {
		return nil, fmt.Errorf("SignPublicSSHCertificate: %w", err)
	}

	ssh := vc.Client.SSHWithMountPoint("ssh")
	secret, err := ssh.SignKey("signer", reqMap)
	if err != nil {
		return nil, fmt.Errorf("SignPublicSSHCertificate: %w", err)
	}
	return []byte(secret.Data["signed_key"].(string)), nil
}

func (vc *vaultClient) GetSafeServersByTenant(tenantUUID iam.TenantUUID,
	serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role:       ServersQueryRole,
		TenantUUID: tenantUUID,
	})
	if err != nil {
		return nil, err
	}
	requestPath := fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/query_server", tenantUUID)

	servers, err := vc.getSafeServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.GetServersByTenant: %w", err)
	}
	return servers, nil
}

func (vc *vaultClient) GetSafeServers(serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role: ServersQueryRole,
	})
	if err != nil {
		return nil, err
	}
	requestPath := "/v1/auth/flant_iam_auth/query_server"
	servers, err := vc.getSafeServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.GetServers: %w", err)
	}
	return servers, nil
}

// getServers get servers by path, adding params to request, it is wrapping over queryServers method
func (vc *vaultClient) getServers(requestPath string, serverIdentifiers []string, labelSelector string) ([]ext.Server, error) {
	var vaultServersResponse struct {
		Data struct {
			Servers []ext.Server `json:"servers"`
		} `json:"data"`
	}
	err := vc.queryServers(requestPath, serverIdentifiers, labelSelector, &vaultServersResponse)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.getServers: %w", err)
	}

	servers := vaultServersResponse.Data.Servers

	return servers, nil
}

// queryServers query vault by given path and args, and parse result into given vaultServersResponsePtr
func (vc *vaultClient) queryServers(requestPath string, serverIdentifiers []string, labelSelector string,
	vaultServersResponsePtr interface{}) error {
	if len(serverIdentifiers) > 0 && labelSelector != "" {
		return fmt.Errorf("queryServers: only serverIdentifiers or labelSelector must be set")
	}
	var params url.Values
	if len(serverIdentifiers) > 0 {
		params = url.Values{"names": []string{strings.Join(serverIdentifiers, ",")}}
	}
	if labelSelector != "" {
		params = url.Values{"labelSelector": []string{labelSelector}}
	}

	vaultServersResponseBytes, err := vc.makeRequest("GET", requestPath, params, nil)
	if err != nil {
		return fmt.Errorf("VaultClient.queryServers: %w", err)
	}

	err = json.Unmarshal(vaultServersResponseBytes, vaultServersResponsePtr)
	if err != nil {
		return fmt.Errorf("VaultClient.queryServers: %w", err)
	}
	return nil
}

// getServers get servers by path, adding params to request, returns SafeServer
func (vc *vaultClient) getSafeServers(requestPath string, serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	var vaultServersResponse struct {
		Data struct {
			Servers []auth_ext.SafeServer `json:"servers"`
		} `json:"data"`
	}

	err := vc.queryServers(requestPath, serverIdentifiers, labelSelector, &vaultServersResponse)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.getServers: %w", err)
	}

	servers := vaultServersResponse.Data.Servers

	return servers, nil
}

func (vc *vaultClient) GetTenantByUUID(tenantUUID string) (*iam.Tenant, error) {
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role:       TenantReadAuthRole,
		TenantUUID: tenantUUID,
	})
	if err != nil {
		return nil, err
	}
	vaultTenantsResponseBytes, err := vc.makeRequest("GET", "/v1/auth/flant_iam_auth/tenant/"+tenantUUID, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("getTenantByUUID: %w", err)
	}
	var vaultTenantResponse struct {
		Data struct {
			Tenant iam.Tenant `json:"tenant"`
		} `json:"data"`
	}
	err = json.Unmarshal(vaultTenantsResponseBytes, &vaultTenantResponse)
	if err != nil {
		return nil, fmt.Errorf("getTenantByUUID: %w", err)
	}
	return &vaultTenantResponse.Data.Tenant, nil
}

func (vc *vaultClient) GetProjectByUUID(tenantUUID string, projectUUID string) (*iam.Project, error) {
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role:       TenantReadAuthRole,
		TenantUUID: tenantUUID,
	})
	if err != nil {
		return nil, err
	}
	vaultTenantsResponseBytes, err := vc.makeRequest("GET", "/v1/auth/flant_iam_auth/tenant/"+tenantUUID+
		"/project/"+projectUUID, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("getProjectByUUIDs: %w", err)
	}
	var vaultProjectResponse struct {
		Data struct {
			Project iam.Project `json:"project"`
		} `json:"data"`
	}
	err = json.Unmarshal(vaultTenantsResponseBytes, &vaultProjectResponse)
	if err != nil {
		return nil, fmt.Errorf("getProjectByUUIDs: %w", err)
	}
	return &vaultProjectResponse.Data.Project, nil
}

// GetTenantByIdentifier wraps GetTenants
func (vc *vaultClient) GetTenantByIdentifier(tenantIdentifier string) (*iam.Tenant, error) {
	tenants, err := vc.GetTenants()
	if err != nil {
		return nil, err
	}
	var tenant *iam.Tenant
	for _, t := range tenants {
		if t.Identifier == tenantIdentifier {
			tenant = &t
			break
		}
	}
	if tenant == nil {
		return nil, fmt.Errorf("tenant:%w", consts.ErrNotFound)
	}
	return tenant, nil
}

// GetProjectByIdentifier wraps GetProjects
func (vc *vaultClient) GetProjectByIdentifier(tenantUUID iam.TenantUUID, projectIdentifier string) (*auth.Project, error) {
	projects, err := vc.GetProjects(tenantUUID)
	if err != nil {
		return nil, err
	}
	var project *auth.Project
	for _, p := range projects {
		if p.Identifier == projectIdentifier {
			project = &p
			break
		}
	}
	if project == nil {
		return nil, fmt.Errorf("project:%w", consts.ErrNotFound)
	}
	return project, nil
}

func (vc *vaultClient) RegisterServer(server ext.Server) (ext.ServerUUID, iam.MultipassJWT, error) {
	path := fmt.Sprintf("/v1/flant_iam/tenant/%s/project/%s/register_server", server.TenantUUID, server.ProjectUUID)
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role:        ServersRegisterRole,
		TenantUUID:  server.TenantUUID,
		ProjectUUID: server.ProjectUUID,
	})
	if err != nil {
		return "", "", err
	}
	data := map[string]interface{}{
		"identifier":  server.Identifier,
		"labels":      server.Labels,
		"annotations": server.Annotations,
	}
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return "", "", err
	}
	vaultRegisterServerResponseBytes, err := vc.makeRequest("POST", path, nil, bodyBytes)
	if err != nil {
		return "", "", err
	}

	var vaultRegisterServerResponse struct {
		Data struct {
			MultipassJWT string `json:"multipassJWT"`
			ServerUUID   string `json:"uuid"`
		} `json:"data"`
	}
	err = json.Unmarshal(vaultRegisterServerResponseBytes, &vaultRegisterServerResponse)
	if err != nil {
		return "", "", fmt.Errorf("RegisterServer: %w", err)
	}

	return vaultRegisterServerResponse.Data.ServerUUID, vaultRegisterServerResponse.Data.MultipassJWT, nil
}

func (vc *vaultClient) UpdateServerConnectionInfo(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
	serverUUID ext.ServerUUID, connInfo ext.ConnectionInfo) (*ext.Server, error) {
	err := vc.checkForRolesAndUpdateClient(authdapi.RoleWithClaim{
		Role:        ServersRegisterRole,
		TenantUUID:  tenantUUID,
		ProjectUUID: projectUUID,
	})
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/v1/flant_iam/tenant/%s/project/%s/server/%s/connection_info", tenantUUID, projectUUID, serverUUID)
	data := map[string]interface{}{
		"hostname":      connInfo.Hostname,
		"port":          connInfo.Port,
		"jump_hostname": connInfo.JumpHostname,
		"jump_port":     connInfo.JumpPort,
	}
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	updateServerConnectionInfoBytes, err := vc.makeRequest("POST", path, nil, bodyBytes)
	if err != nil {
		return nil, err
	}

	var updateServerConnectionInfoResponse struct {
		Data struct {
			Server ext.Server `json:"server"`
		} `json:"data"`
	}
	err = json.Unmarshal(updateServerConnectionInfoBytes, &updateServerConnectionInfoResponse)
	if err != nil {
		return nil, fmt.Errorf("RegisterServer: %w", err)
	}

	return &(updateServerConnectionInfoResponse.Data.Server), nil
}
