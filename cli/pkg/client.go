package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	vault_api "github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/cli/internal/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	auth_ext "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

// wrap requests to vault
type VaultClient struct {
	*vault_api.Client // authorized client
}

func (vs VaultClient) makeRequest(method, requestPath string, params url.Values, bodyBytes []byte) ([]byte, error) {
	req := vs.NewRequest(method, requestPath)
	if params != nil {
		req.Params = params
	}
	if len(bodyBytes) > 0 {
		req.BodyBytes = bodyBytes
	}
	resp, err := vs.Client.RawRequest(req)
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

func (vs *VaultClient) GetTenants() ([]iam.Tenant, error) {
	vaultTenantsResponseBytes, err := vs.makeRequest("LIST", "/v1/auth/flant_iam_auth/tenant", nil, nil)
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

func (vs *VaultClient) GetProjects(tenantUUID iam.TenantUUID) ([]auth.Project, error) {
	vaultProjectsResponseBytes, err := vs.makeRequest("LIST",
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
func (vs *VaultClient) GetServersByTenantAndProject(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
	serverIdentifiers []string, labelSelector string) ([]ext.Server, error) {

	requestPath := fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project/%s/query_server", tenantUUID, projectUUID)

	servers, err := vs.getServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.GetServersByTenantAndProject: %w", err)
	}
	return servers, nil
}

func (vs *VaultClient) GetUser() (*auth.User, error) {
	vaultResponseBytes, err := vs.makeRequest("GET", "/v1/auth/flant_iam_auth/vst_owner", nil, nil)
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

func (vs *VaultClient) SignPublicSSHCertificate(vaultReq model.VaultSSHSignRequest) ([]byte, error) {
	var reqMap map[string]interface{}
	data, _ := json.Marshal(vaultReq)
	err := json.Unmarshal(data, &reqMap)
	if err != nil {
		return nil, fmt.Errorf("SignPublicSSHCertificate: %w", err)
	}

	ssh := vs.Client.SSHWithMountPoint("ssh")
	secret, err := ssh.SignKey("signer", reqMap)
	if err != nil {
		return nil, fmt.Errorf("SignPublicSSHCertificate: %w", err)
	}
	return []byte(secret.Data["signed_key"].(string)), nil
}

func (vs *VaultClient) GetSafeServersByTenant(tenantUUID iam.TenantUUID,
	serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	requestPath := fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/query_server", tenantUUID)

	servers, err := vs.getSafeServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.GetServersByTenant: %w", err)
	}
	return servers, nil
}

func (vs *VaultClient) GetSafeServers(serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	requestPath := "/v1/auth/flant_iam_auth/query_server"
	servers, err := vs.getSafeServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.GetServers: %w", err)
	}
	return servers, nil
}

// getServers get servers by path, adding params to request
func (vs *VaultClient) getServers(requestPath string, serverIdentifiers []string, labelSelector string) ([]ext.Server, error) {
	var vaultServersResponse struct {
		Data struct {
			Servers []ext.Server `json:"servers"`
		} `json:"data"`
	}

	err := vs.queryServers(requestPath, serverIdentifiers, labelSelector, &vaultServersResponse)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.getServers: %w", err)
	}

	servers := vaultServersResponse.Data.Servers

	return servers, nil
}

// queryServers query vault by given path and args, and parse result into given vaultServersResponsePtr
func (vs *VaultClient) queryServers(requestPath string, serverIdentifiers []string, labelSelector string,
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

	vaultServersResponseBytes, err := vs.makeRequest("GET", requestPath, params, nil)
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
func (vs *VaultClient) getSafeServers(requestPath string, serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	var vaultServersResponse struct {
		Data struct {
			Servers []auth_ext.SafeServer `json:"servers"`
		} `json:"data"`
	}

	err := vs.queryServers(requestPath, serverIdentifiers, labelSelector, &vaultServersResponse)
	if err != nil {
		return nil, fmt.Errorf("VaultClient.getServers: %w", err)
	}

	servers := vaultServersResponse.Data.Servers

	return servers, nil
}

func (vs *VaultClient) GetTenantByUUID(tenantUUID string) (*iam.Tenant, error) {
	vaultTenantsResponseBytes, err := vs.makeRequest("GET", "/v1/auth/flant_iam_auth/tenant/"+tenantUUID, nil, nil)
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

func (vs *VaultClient) GetProjectByUUIDs(tenantUUID string, projectUUID string) (*iam.Project, error) {
	vaultTenantsResponseBytes, err := vs.makeRequest("GET", "/v1/auth/flant_iam_auth/tenant/"+tenantUUID+
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

func (vs *VaultClient) GetTenantByIdentifier(tenantIdentifier string) (*iam.Tenant, error) {
	tenants, err := vs.GetTenants()
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

func (vs *VaultClient) GetProjectByIdentifier(tenantUUID iam.TenantUUID, projectIdentifier string) (*auth.Project, error) {
	projects, err := vs.GetProjects(tenantUUID)
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

func (vs *VaultClient) RegisterServer(server ext.Server) (ext.ServerUUID, iam.MultipassJWT, error) {
	path := fmt.Sprintf("/v1/flant_iam/tenant/%s/project/%s/register_server", server.TenantUUID, server.ProjectUUID)
	data := map[string]interface{}{
		"identifier":  server.Identifier,
		"labels":      server.Labels,
		"annotations": server.Annotations,
	}
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return "", "", err
	}
	vaultRegisterServerResponseBytes, err := vs.makeRequest("POST", path, nil, bodyBytes)
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

func (vs *VaultClient) UpdateServerConnectionInfo(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
	serverUUID ext.ServerUUID, connInfo ext.ConnectionInfo) (*ext.Server, error) {
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
	updateServerConnectionInfoBytes, err := vs.makeRequest("POST", path, nil, bodyBytes)
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
