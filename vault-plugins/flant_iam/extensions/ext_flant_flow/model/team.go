package model

import (
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const TeamType = "team" // also, memdb schema name

type Team struct {
	memdb.ArchivableImpl

	UUID           TeamUUID `json:"uuid"` // PK
	Version        string   `json:"resource_version"`
	Identifier     string   `json:"identifier"`
	TeamType       string   `json:"team_type"` // it is unchangeable
	ParentTeamUUID string   `json:"parent_team_uuid"`

	// TODO how to deal with?
	// 1) only autocreate and autodelet?
	// 2) something else?
	Groups []LinkedGroup
}

type LinkedGroup struct {
	GroupUUID iam_model.GroupUUID
	Type      string
}

func (u *Team) ObjType() string {
	return TeamType
}

func (u *Team) ObjId() string {
	return u.UUID
}
