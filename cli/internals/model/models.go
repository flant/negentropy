package model

import (
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type ServerFilter struct {
	TenantIdentifiers  []string
	ProjectIdentifiers []string
	LabelSelectors     []string
	ServerIdentifiers  []string
}

type ServerList struct {
	Tenants  map[iam.TenantUUID]iam.Tenant
	Projects map[iam.ProjectUUID]iam.Project
	Servers  []ext.Server
}

type VaultSSHSignRequest struct {
	PublicKey       string `json:"public_key"`
	ValidPrincipals string `json:"valid_principals"`
}
