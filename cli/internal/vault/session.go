package vault

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	vault_api "github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/authd"
	authdapi "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/cli/internal/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
	auth_ext "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
)

// wrap requests to vault
type VaultSession struct {
	Client *vault_api.Client
}

func NewVaultSession() VaultSession {
	authdClient := authd.NewAuthdClient("/run/authd.sock")

	req := authdapi.NewLoginRequest().
		WithRoles(authdapi.NewRoleWithClaim("*", map[string]string{})).
		WithServerType(authdapi.AuthServer)

	err := authdClient.OpenVaultSession(req)
	if err != nil {
		panic(err)
	}

	vaultClient, err := authdClient.NewVaultClient()
	if err != nil {
		panic(err)
	}

	return VaultSession{
		Client: vaultClient,
	}
}

func (vs *VaultSession) makeRequest(method, requestPath string, params url.Values) ([]byte, error) {
	req := vs.Client.NewRequest(method, requestPath)
	if params != nil {
		req.Params = params
	}
	resp, err := vs.Client.RawRequest(req)
	defer resp.Body.Close()

	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}

func (vs *VaultSession) getTenants() ([]iam.Tenant, error) {
	vaultTenantsResponseBytes, err := vs.makeRequest("LIST", "/v1/auth/flant_iam_auth/tenant", nil)
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

func (vs *VaultSession) getProjects(tenantUUID iam.TenantUUID) ([]auth.SafeProject, error) {
	vaultProjectsResponseBytes, err := vs.makeRequest("LIST",
		fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project", tenantUUID), nil)
	if err != nil {
		return nil, fmt.Errorf("getProjects: %w", err)
	}
	var vaultProjectsResponse struct {
		Data struct {
			Projects []auth.SafeProject `json:"projects"`
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
func (vs *VaultSession) GetServersByTenantAndProject(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID,
	serverIdentifiers []string, labelSelector string) ([]ext.Server, error) {

	requestPath := fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project/%s/query_server", tenantUUID, projectUUID)

	servers, err := vs.getServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultSession.GetServersByTenantAndProject: %w", err)
	}
	return servers, nil
}

//func (vs *VaultSession) GetServerToken(server ext.Server) (string, error) {
//	vaultServerTokenResponseBytes, err := vs.makeRequest("GET",
//		fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project/%s/server/%s",
//			server.TenantUUID, server.ProjectUUID, server.UUID), nil)
//	if err != nil {
//		return "", err
//	}
//
//	var vaultServerTokenResponse struct {
//		Data struct {
//			Token string `json:"token"`
//		} `json:"data"`
//	}
//
//	err = json.Unmarshal(vaultServerTokenResponseBytes, &vaultServerTokenResponse)
//	if err != nil {
//		return "", err
//	}
//
//	return vaultServerTokenResponse.Data.Token, nil
//}

func (vs *VaultSession) GetUser() (*auth.User, error) {
	vaultResponseBytes, err := vs.makeRequest("GET", "/v1/auth/flant_iam_auth/multipass_owner", nil)
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

func (vs *VaultSession) SignPublicSSHCertificate(vaultReq model.VaultSSHSignRequest) ([]byte, error) {
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

func (vs *VaultSession) GetSafeServersByTenant(tenantUUID iam.TenantUUID,
	serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	requestPath := fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/query_server", tenantUUID)

	servers, err := vs.getSafeServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultSession.GetServersByTenant: %w", err)
	}
	return servers, nil
}

func (vs *VaultSession) GetSafeServers(serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	requestPath := "/v1/auth/flant_iam_auth/query_server"
	servers, err := vs.getSafeServers(requestPath, serverIdentifiers, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("VaultSession.GetServers: %w", err)
	}
	return servers, nil
}

// getServers get servers by path, adding params to request
func (vs *VaultSession) getServers(requestPath string, serverIdentifiers []string, labelSelector string) ([]ext.Server, error) {
	var vaultServersResponse struct {
		Data struct {
			Servers []ext.Server `json:"servers"`
		} `json:"data"`
	}

	err := vs.queryServers(requestPath, serverIdentifiers, labelSelector, &vaultServersResponse)
	if err != nil {
		return nil, fmt.Errorf("VaultSession.getServers: %w", err)
	}

	servers := vaultServersResponse.Data.Servers

	return servers, nil
}

// queryServers query vault by given path and args, and parse result into given vaultServersResponsePtr
func (vs *VaultSession) queryServers(requestPath string, serverIdentifiers []string, labelSelector string,
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

	vaultServersResponseBytes, err := vs.makeRequest("GET", requestPath, params)
	if err != nil {
		return fmt.Errorf("VaultSession.queryServers: %w", err)
	}

	err = json.Unmarshal(vaultServersResponseBytes, vaultServersResponsePtr)
	if err != nil {
		return fmt.Errorf("VaultSession.queryServers: %w", err)
	}
	return nil
}

// getServers get servers by path, adding params to request, returns SafeServer
func (vs *VaultSession) getSafeServers(requestPath string, serverIdentifiers []string, labelSelector string) ([]auth_ext.SafeServer, error) {
	var vaultServersResponse struct {
		Data struct {
			Servers []auth_ext.SafeServer `json:"servers"`
		} `json:"data"`
	}

	err := vs.queryServers(requestPath, serverIdentifiers, labelSelector, &vaultServersResponse)
	if err != nil {
		return nil, fmt.Errorf("VaultSession.getServers: %w", err)
	}

	servers := vaultServersResponse.Data.Servers

	return servers, nil
}

func (vs *VaultSession) getTenantByUUID(tenantUUID string) (*iam.Tenant, error) {
	vaultTenantsResponseBytes, err := vs.makeRequest("GET", "/v1/auth/flant_iam_auth/tenant/"+tenantUUID, nil)
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

func (vs *VaultSession) getProjectByUUIDs(tenantUUID string, projectUUID string) (*iam.Project, error) {
	vaultTenantsResponseBytes, err := vs.makeRequest("GET", "/v1/auth/flant_iam_auth/tenant/"+tenantUUID+
		"/project/"+projectUUID, nil)
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
