package model

import (
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const TenantType = "tenant" // also, memdb schema name

type Tenant struct {
	memdb.ArchiveMark

	UUID       TenantUUID `json:"uuid"` // PK
	Version    string     `json:"resource_version"`
	Identifier string     `json:"identifier"`
	Language   string     `json:"language"`

	Origin consts.ObjectOrigin `json:"origin,omitempty"`

	FeatureFlags []TenantFeatureFlag `json:"feature_flags"`
}

func (t *Tenant) ObjType() string {
	return TenantType
}

func (t *Tenant) ObjId() string {
	return t.UUID
}
