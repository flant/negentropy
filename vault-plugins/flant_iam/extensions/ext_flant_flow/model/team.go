package model

const TeamType = "team" // also, memdb schema name

type Team struct {
	UUID           TeamUUID `json:"uuid"` // PK
	Version        string   `json:"resource_version"`
	Identifier     string   `json:"identifier"`
	TeamType       string   `json:"team_type"`
	ParentTeamUUID string   `json:"parent_team_uuid"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (u *Team) IsDeleted() bool {
	return u.ArchivingTimestamp != 0
}

func (u *Team) ObjType() string {
	return TeamType
}

func (u *Team) ObjId() string {
	return u.UUID
}
