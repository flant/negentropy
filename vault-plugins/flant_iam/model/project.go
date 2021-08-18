package model

const ProjectType = "project" // also, memdb schema name

type Project struct {
	UUID       ProjectUUID `json:"uuid"` // PK
	TenantUUID TenantUUID  `json:"tenant_uuid"`
	Version    string      `json:"resource_version"`
	Identifier string      `json:"identifier"`

	FeatureFlags []FeatureFlag `json:"feature_flags"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (p *Project) IsDeleted() bool {
	return p.ArchivingTimestamp != 0
}

func (u *Project) ObjType() string {
	return ProjectType
}

func (u *Project) ObjId() string {
	return u.UUID
}
