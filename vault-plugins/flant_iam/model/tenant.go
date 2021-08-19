package model

const TenantType = "tenant" // also, memdb schema name

type Tenant struct {
	UUID       TenantUUID `json:"uuid"` // PK
	Version    string     `json:"resource_version"`
	Identifier string     `json:"identifier"`

	FeatureFlags []TenantFeatureFlag `json:"feature_flags"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (t *Tenant) IsDeleted() bool {
	return t.ArchivingTimestamp != 0
}

func (t *Tenant) ObjType() string {
	return TenantType
}

func (t *Tenant) ObjId() string {
	return t.UUID
}
