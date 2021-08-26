package model

import "encoding/json"

const ServerType = "server" // also, memdb schema name

type ServerUUID = string

type Server struct {
	UUID          ServerUUID `json:"uuid"` // ID
	TenantUUID    string     `json:"tenant_uuid"`
	ProjectUUID   string     `json:"project_uuid"`
	Version       string     `json:"resource_version"`
	Identifier    string     `json:"identifier"`
	MultipassUUID string     `json:"multipass_uuid"`

	Fingerprint string            `json:"fingerprint"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	ConnectionInfo ConnectionInfo `json:"connection_info"`
}

type ConnectionInfo struct {
	Hostname     string `json:"hostname"`
	Port         string `json:"port"`
	JumpHostname string `json:"jump_hostname"`
	JumpPort     string `json:"jump_port"`
}

func (c *ConnectionInfo) FillDefaultPorts() {
	if c.Port == "" {
		c.Port = "22"
	}
	if c.JumpHostname != "" && c.JumpPort == "" {
		c.JumpPort = "22"
	}
}

func (u *Server) ObjType() string {
	return ServerType
}

func (u *Server) ObjId() string {
	return u.UUID
}

func (u *Server) AsMap() map[string]interface{} {
	var res map[string]interface{}

	data, _ := json.Marshal(u)

	_ = json.Unmarshal(data, &res)

	return res
}
