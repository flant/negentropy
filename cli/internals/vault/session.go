package vault

import (
	"bytes"
	"encoding/json"
	"fmt"

	vault_api "github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/authd"
	authdapi "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/cli/internals/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
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

func (vs *VaultSession) makeRequest(method, requestPath string) ([]byte, error) {
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

func (vs *VaultSession) getTenants() ([]iam.Tenant, error) { // TODO UP
	vaultTenantsResponseBytes, err := vs.makeRequest("LIST", "/v1/auth/flant_iam_auth/tenant")
	if err != nil {
		return []iam.Tenant{}, err
	}

	var vaultTenantsResponse struct {
		Data struct {
			Tenants []iam.Tenant `json:"tenants"`
		} `json:"data"`
	}
	err = json.Unmarshal(vaultTenantsResponseBytes, &vaultTenantsResponse)
	if err != nil {
		return []iam.Tenant{}, err
	}

	return vaultTenantsResponse.Data.Tenants, nil
}

func (vs *VaultSession) getProjects(tenantUUID iam.TenantUUID) ([]iam.Project, error) { // TODO UP
	vaultProjectsResponseBytes, err := vs.makeRequest("LIST",
		fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project", tenantUUID))
	if err != nil {
		return []iam.Project{}, err
	}

	var vaultProjectsResponse struct {
		Data struct {
			Projects []iam.Project `json:"projects"`
		} `json:"data"`
	}
	err = json.Unmarshal(vaultProjectsResponseBytes, &vaultProjectsResponse)
	if err != nil {
		return []iam.Project{}, err
	}

	projects := vaultProjectsResponse.Data.Projects
	return projects, nil
}

// TODO filter
// TODO warnings
func (vs *VaultSession) GetServers(tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID) ([]ext.Server, error) {
	var vaultServersResponse struct {
		Data struct {
			Servers []ext.Server `json:"servers"`
		} `json:"data"`
	}
	requestPath := fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project/%s/query_server", tenantUUID, projectUUID)

	vaultServersResponseBytes, err := vs.makeRequest("GET", requestPath)
	if err != nil {
		return []ext.Server{}, err
	}

	err = json.Unmarshal(vaultServersResponseBytes, &vaultServersResponse)
	if err != nil {
		return []ext.Server{}, err
	}

	servers := vaultServersResponse.Data.Servers

	return servers, nil
}

func (vs *VaultSession) GetServerToken(server ext.Server) (string, error) {
	vaultServerTokenResponseBytes, err := vs.makeRequest("GET",
		fmt.Sprintf("/v1/auth/flant_iam_auth/tenant/%s/project/%s/server/%s",
			server.TenantUUID, server.ProjectUUID, server.UUID))
	if err != nil {
		return "", err
	}

	var vaultServerTokenResponse struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	err = json.Unmarshal(vaultServerTokenResponseBytes, &vaultServerTokenResponse)
	if err != nil {
		return "", err
	}

	return vaultServerTokenResponse.Data.Token, nil
}

func (vs *VaultSession) GetUser() (*auth.User, error) {
	vaultResponseBytes, err := vs.makeRequest("GET", "/v1/auth/flant_iam_auth/multipass_owner")
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

func (vs *VaultSession) getTenantByIdentifier(tenantIdentifier string) (iam.Tenant, error) {
	tenants, err := vs.getTenants()
	if err != nil {
		return iam.Tenant{}, err
	}

	for _, tenant := range tenants {
		if tenant.Identifier == tenantIdentifier {
			return tenant, nil
		}
	}

	return iam.Tenant{}, nil
}

func (vs *VaultSession) SignPublicSSHCertificate(vaultReq model.VaultSSHSignRequest) []byte {
	var reqMap map[string]interface{}
	data, _ := json.Marshal(vaultReq)
	json.Unmarshal(data, &reqMap)

	ssh := vs.Client.SSHWithMountPoint("ssh")
	secret, err := ssh.SignKey("signer", reqMap)
	if err != nil {
		panic(err)
	}

	return []byte(secret.Data["signed_key"].(string))
}
