package api

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"
)

type Tenant struct {
	UUID       string
	Identifier string
}

type Project struct {
	UUID       string
	Identifier string
	Tenant     *Tenant
}

type Server struct {
	UUID        string
	Identifier  string
	Project     *Project
	Version     int64
	JWTManifest string
	Manifest    ServerManifest
}

type ServerManifest struct {
	Hostname    string
	Port        int64
	Fingerprint string // ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBD50RKBgjQ7YlvVqNosJ3ovmaNyor+riouDuZvwgOXARr2WSIf4tt9DR1k3X7k1+oTZJWtE7w3GJnDMzDaiw7gc=
	Bastion     *Server
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

func (s *Server) GenerateUserPrincipal(user User) string {
	hash := sha256.Sum256([]byte(s.UUID + user.UUID))
	return fmt.Sprintf("%x", hash)
}

func (s *Server) RenderKnownHostsRow() string {
	// TODO Shouldn't it be in session.go?
	if s.Manifest.Port == 22 {
		return fmt.Sprintf("%s %s\n", s.Manifest.Hostname, s.Manifest.Fingerprint)
	} else {
		return fmt.Sprintf("[%s]:%d %s\n", s.Manifest.Hostname, s.Manifest.Port, s.Manifest.Fingerprint)
	}
}

func (s *Server) RenderSSHConfigEntry() string {
	// TODO Shouldn't it be in session.go?
	entryBuffer := bytes.Buffer{}

	tmpl, err := template.New("ssh_config_entry").Parse(`
Host {{.Project.Identifier}}.{{.Identifier}}
  ForwardAgent yes
  Hostname {{.Manifest.Hostname}}
{{- if .Manifest.Port }}
  Port {{.Manifest.Port}}
{{- end }}
{{- if .Manifest.Bastion }}
  ProxyCommand ssh {{.Manifest.Bastion.Project.Identifier}}.{{.Manifest.Bastion.Identifier}} -W %h:%p
{{- end }}

`)
	if err != nil {
		panic(err)
	}

	err = tmpl.Execute(&entryBuffer, s)
	if err != nil {
		panic(err)
	}
	return entryBuffer.String()
}
