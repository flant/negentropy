package model

import (
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const GroupType = "group" // also, memdb schema name

type Group struct {
	memdb.ArchiveMark

	UUID           GroupUUID  `json:"uuid"` // PK
	TenantUUID     TenantUUID `json:"tenant_uuid"`
	Version        string     `json:"resource_version"`
	Identifier     string     `json:"identifier"`
	FullIdentifier string     `json:"full_identifier"`

	Users           []UserUUID           `json:"users"`
	Groups          []GroupUUID          `json:"groups"`
	ServiceAccounts []ServiceAccountUUID `json:"service_accounts"`
	Members         []MemberNotation     `json:"members"`

	Origin consts.ObjectOrigin `json:"origin"`

	Extensions map[consts.ObjectOrigin]*Extension `json:"-"`
}

func (g *Group) ObjType() string {
	return GroupType
}

func (g *Group) ObjId() string {
	return g.UUID
}

// FixMembers remove from members invalid links, if some removed, returns true
func (g *Group) FixMembers() bool {
	return FixMembers(&g.Members, g.Users, g.Groups, g.ServiceAccounts)
}
