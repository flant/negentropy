package model

import "github.com/flant/negentropy/vault-plugins/shared/memdb"

const GroupType = "group" // also, memdb schema name

type Group struct {
	memdb.ArchivableImpl

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
}

func (g *Group) ObjType() string {
	return GroupType
}

func (g *Group) ObjId() string {
	return g.UUID
}

// FixMembers remove from members invalid links, if some removed, returns true
func (g *Group) FixMembers() bool {
	if len(g.Members) == len(g.Users)+len(g.Groups)+len(g.ServiceAccounts) {
		return false
	}
	buildSet := func(uuids []string) map[string]struct{} {
		result := map[string]struct{}{}
		for _, uuid := range uuids {
			result[uuid] = struct{}{}
		}
		return result
	}
	membersSuperSet := map[string]map[string]struct{}{
		UserType:           buildSet(g.Users),
		ServiceAccountType: buildSet(g.ServiceAccounts),
		GroupType:          buildSet(g.Groups),
	}
	newMembers := make([]MemberNotation, 0, len(g.Members))

	fixed := false
	for _, m := range g.Members {
		if _, ok := membersSuperSet[m.Type][m.UUID]; ok {
			newMembers = append(newMembers, m)
		} else {
			fixed = true
		}
	}
	g.Members = newMembers
	return fixed
}
