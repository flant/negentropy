package model

const GroupType = "group" // also, memdb schema name

type Group struct {
	UUID           GroupUUID  `json:"uuid"` // PK
	TenantUUID     TenantUUID `json:"tenant_uuid"`
	Version        string     `json:"resource_version"`
	Identifier     string     `json:"identifier"`
	FullIdentifier string     `json:"full_identifier"`

	Users           []UserUUID           `json:"-"`
	Groups          []GroupUUID          `json:"-"`
	ServiceAccounts []ServiceAccountUUID `json:"-"`
	Members         []MemberNotation     `json:"members"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"-"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (g *Group) IsDeleted() bool {
	return g.ArchivingTimestamp != 0
}

func (g *Group) ObjType() string {
	return GroupType
}

func (g *Group) ObjId() string {
	return g.UUID
}
