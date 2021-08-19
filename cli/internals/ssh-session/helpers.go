package ssh_session

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"html/template"

	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func GenerateUserPrincipal(s ext.Server, user iam.User) string {
	hash := sha256.Sum256([]byte(s.UUID + user.UUID))
	return fmt.Sprintf("%x", hash)
}

func RenderKnownHostsRow(s ext.Server) string {
	if s.ConnectionInfo.Port == "22" {
		return fmt.Sprintf("%s %s\n", s.ConnectionInfo.Hostname, s.Fingerprint)
	} else {
		return fmt.Sprintf("[%s]:%d %s\n", s.ConnectionInfo.Hostname, s.ConnectionInfo.Port, s.Fingerprint)
	}
}

func RenderSSHConfigEntry(project iam.Project, s ext.Server, user iam.User) string {
	entryBuffer := bytes.Buffer{}
	tmpl, err := template.New("ssh_config_entry").Parse(`
Host {{.Project.Identifier}}.{{.Server.Identifier}}
  ForwardAgent yes
  User {{.User.FullIdentifier}}
  Hostname {{.Server.ConnectionInfo.Hostname}}
{{- if .Server.ConnectionInfo.Port }}
  Port {{.Server.ConnectionInfo.Port}}
{{- end }}
{{- if .Server.ConnectionInfo.JumpHostname }}
  ProxyCommand ssh {{.ServerConnectionInfo.JumpHostname}} -W %h:%p
{{- end }}

`)
	if err != nil {
		panic(err)
	}

	context := struct {
		Server  ext.Server
		User    iam.User
		Project iam.Project
	}{
		Server:  s,
		User:    user,
		Project: project,
	}

	err = tmpl.Execute(&entryBuffer, context)
	if err != nil {
		panic(err)
	}
	return entryBuffer.String()
}
