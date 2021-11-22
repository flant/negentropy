package model

import "github.com/flant/negentropy/vault-plugins/shared/memdb"

const ProjectType = "project" // also, memdb schema name

type Project struct {
	memdb.ArchivableImpl

	UUID       ProjectUUID `json:"uuid"` // PK
	TenantUUID TenantUUID  `json:"tenant_uuid"`
	Version    string      `json:"resource_version"`
	Identifier string      `json:"identifier"`

	FeatureFlags []FeatureFlag `json:"feature_flags"`
}

func (p *Project) ObjType() string {
	return ProjectType
}

func (p *Project) ObjId() string {
	return p.UUID
}
