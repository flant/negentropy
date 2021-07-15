package vault

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"main/pkg/api"
	"net/http"
)

type VaultSession struct {
	SessionToken   string
	UserIdentifier string // где выдрать?
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

type VaultSSHSignResponse struct {
	Data VaultSSHSignResponseData `json:"data"`
}

type VaultSSHSignResponseData struct {
	SignedKey string `json:"signed_key"`
}

func (sf *ServerFilter) RenderURIArgs() string {
	// ?name[]=db1&name[]=db-2&...
	// labelselector=...
	return ""
}

func (vs *VaultSession) Init() {
	// подключиться в authd и получить токен
	// запустить рутинку, которая будет обновлять SessionToken через vault
	vs.SessionToken = "s.eGpFgdmlHgvmuU2Bd9QDyAa4"
}

func (vs *VaultSession) GetUser() api.User {
	// достать из vault инфу про текущего юзера
	return api.User{UUID: "uuu", Identifier: "a.polovov"}
}

func (vs *VaultSession) getTenantByIdentifier(identifier string) api.Tenant {
	// LIST /tenant -> ищем наш
	//tenant := api.Tenant{
	//	Identifier: identifier,
	//	UUID: <uuid>
	//}
	// return tenant
	return api.Tenant{}
}

func (vs *VaultSession) getProjectsByTenant(tenant *api.Tenant) []api.Project {
	// LIST /tenant/<tenant.UUID>/project
	// каждый проект:
	// упаковать в api.Project
	// Сослаться на тенант
	// добавить в массив и потом вернуть
	return []api.Project{}
}

func (vs *VaultSession) getServerManifest(server api.Server) api.ServerManifest {
	// GET /tenant/<server.Tenant.UUID>/project/<server.Project.UUID>/server/<server.UUID>
	// return manifest
	return api.ServerManifest{}
}

func (vs *VaultSession) QueryServer(filter ServerFilter) api.ServerList {
	// sl.Tenant = vs.getTenantByIdentifier(filter.TenantIdentifier)
	// если в фильтре есть ограничения по проектам:
	//   projects := vs.getProjectsByTenant(&sl.Tenant)
	//   выгрести лишние проекты из ответа по фильтру
	//   для оставшихся проектов: LIST /tenant/<tenant.UUID>/project/<project.UUID>/query_server?<filter.RenderURIArgs()>
	// если ограничений нет, то
	//   LIST /tenant/<tenant.UUID>/query_server?<filter.RenderURIArgs()>

	// == имеем ServerList, осталось заполнить манифесты
	// для каждого сервера server.Manifest = vs.getServerManifest(server)
	// если есть bastion, то как-то его надо заинклудить
	// return serverList
	sl := api.ServerList{
		Tenant: api.Tenant{
			UUID:       "aaa",
			Identifier: "mytenant",
		},
	}
	sl.Projects = append(sl.Projects, api.Project{UUID: "bbb", Identifier: "myproject", Tenant: &sl.Tenant})
	sl.Servers = append(sl.Servers, api.Server{Identifier: "node-1", UUID: "ccc", Project: &sl.Projects[0], Manifest: api.ServerManifest{Hostname: "95.216.34.23", Port: 2202}})
	return sl
}

func (vs *VaultSession) SignPublicSSHCertificate(vaultReq VaultSSHSignRequest) []byte {
	url := "http://127.0.0.1:8200/v1/ssh-client-signer/sign/signer"

	vaultReqJSON, _ := json.Marshal(vaultReq)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(vaultReqJSON))
	req.Header["X-Vault-Token"] = []string{vs.SessionToken}
	req.Header.Set("Content-Type", "application/json")
	resp, _ := client.Do(req)

	var vaultResp VaultSSHSignResponse
	body, _ := ioutil.ReadAll(resp.Body)
	err := json.Unmarshal(body, &vaultResp)
	if err != nil {
		panic(err.Error())
	}
	return []byte(vaultResp.Data.SignedKey)
}
