package ssh_session

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"

	"github.com/flant/negentropy/cli/internal/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extension_server_access/model"
)

func GenerateUserPrincipal(serverUUID, userUUID string) string {
	hash := sha256.Sum256([]byte(serverUUID + userUUID))
	return fmt.Sprintf("%x", hash)
}

func RenderKnownHostsRow(s ext.Server) string {
	if s.ConnectionInfo.Port == "22" {
		return fmt.Sprintf("%s %s\n", s.ConnectionInfo.Hostname, s.Fingerprint)
	} else {
		return fmt.Sprintf("[%s]:%s %s\n", s.ConnectionInfo.Hostname, s.ConnectionInfo.Port, s.Fingerprint)
	}
}

func RenderSSHConfigEntry(project iam.Project, s ext.Server, user auth.User) string {
	entryBuffer := bytes.Buffer{}
	tmpl, err := template.New("ssh_config_entry").Parse(`
Host {{.Project.Identifier}}.{{.Server.Identifier}}
  ForwardAgent yes
  User {{.User.Identifier}}
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
		User    auth.User
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

func SaveToFile(serverList model.ServerList, path string) error {
	data, err := json.Marshal(serverList)
	if err != nil {
		return fmt.Errorf("SaveToFile: %w", err)
	}
	err = ioutil.WriteFile(path, data, 0o644)
	if err != nil {
		return fmt.Errorf("SaveToFile: %w", err)
	}
	return nil
}

func ReadFromFile(path string) (*model.ServerList, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ReadFromFile: %w", err)
	}
	var result model.ServerList
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("ReadFromFile: %w", err)
	}
	return &result, nil
}
