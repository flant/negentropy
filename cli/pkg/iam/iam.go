package iam

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"

	jose "github.com/square/go-jose/v3"
)

type Tenant struct {
	UUID       string `json:"uuid"`
	Identifier string `json:"identifier"`
}

type Project struct {
	UUID       string `json:"uuid"`
	Identifier string `json:"identifier"`
	Tenant     *Tenant
}

type Server struct {
	UUID            string `json:"uuid"`
	TenantUUID      string `json:"tenant_uuid"`
	ProjectUUID     string `json:"project_uuid"`
	Identifier      string `json:"identifier"`
	Project         *Project
	ResourceVersion string `json:"resource_version"`
	Token           string
	SecureManifest  ServerSecureManifest
}

// TODO there are more fields
type ServerSecureManifest struct {
	Fingerprint    string `json:"fingerprint"`
	Identifier     string `json:"identifier"`
	ConnectionInfo struct {
		Hostname string  `json:"hostname"`
		Port     int64   `json:"port"`
		Bastion  *Server // TODO
	} `json:"connection_info"`
}

type ServerList struct {
	Tenant   Tenant
	Projects []Project
	Servers  []Server
}

type User struct {
	UUID       string
	Identifier string
	// ...
}

type TPLContextSSHConfig struct {
	Server *Server
	User   *User
}

func (s *Server) GenerateUserPrincipal(user User) string {
	hash := sha256.Sum256([]byte(s.UUID + user.UUID))
	return fmt.Sprintf("%x", hash)
}

func (s *Server) RenderKnownHostsRow() string {
	// TODO Shouldn't it be in ssh-ssh-session.go?
	if s.SecureManifest.ConnectionInfo.Port == 22 {
		return fmt.Sprintf("%s %s\n", s.SecureManifest.ConnectionInfo.Hostname, s.SecureManifest.Fingerprint)
	} else {
		return fmt.Sprintf("[%s]:%d %s\n", s.SecureManifest.ConnectionInfo.Hostname, s.SecureManifest.ConnectionInfo.Port, s.SecureManifest.Fingerprint)
	}
}

func (s *Server) RenderSSHConfigEntry(user *User) string {
	// TODO Shouldn't it be in ssh-ssh-session.go?

	entryBuffer := bytes.Buffer{}
	tmpl, err := template.New("ssh_config_entry").Parse(`
Host {{.Server.Project.Identifier}}.{{.Server.Identifier}}
  ForwardAgent yes
  User {{.User.Identifier}}
  Hostname {{.Server.SecureManifest.ConnectionInfo.Hostname}}
{{- if .Server.SecureManifest.ConnectionInfo.Port }}
  Port {{.Server.SecureManifest.ConnectionInfo.Port}}
{{- end }}
{{- if .Server.SecureManifest.ConnectionInfo.Bastion }}
  ProxyCommand ssh {{.Server.SecureManifest.ConnectionInfo.Bastion.Project.Identifier}}.{{.Server.SecureManifest.ConnectionInfo.Bastion.Identifier}} -W %h:%p
{{- end }}

`)
	if err != nil {
		panic(err)
	}

	context := TPLContextSSHConfig{
		User:   user,
		Server: s,
	}

	err = tmpl.Execute(&entryBuffer, context)
	if err != nil {
		panic(err)
	}
	return entryBuffer.String()
}

func (s *Server) SetSecureManifest(token string) error {
	// TODO check signature
	jose.ParseSigned(token)

	jwt, err := jose.ParseSigned(token)
	if err != nil {
		return err
	}

	payloadBytes := jwt.UnsafePayloadWithoutVerification()

	err = json.Unmarshal(payloadBytes, &s.SecureManifest)
	if err != nil {
		return err
	}
	return nil
}
