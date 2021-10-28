package model

import iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"

const TeammateType = "teammate" // also, memdb schema name

type Teammate struct {
	iam_model.User

	TeamUUID   TeamUUID   `json:"team_uuid"`
	RoleAtTeam RoleAtTeam `json:"role_at_team"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

func (u *Teammate) IsDeleted() bool {
	return u.ArchivingTimestamp != 0
}

func (u *Teammate) ObjType() string {
	return TeamType
}

func (u *Teammate) ObjId() string {
	return u.UUID
}
