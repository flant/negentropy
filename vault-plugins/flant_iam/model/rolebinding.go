package model

import (
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const RoleBindingType = "role_binding" // also, memdb schema name

type RoleBinding struct {
	memdb.ArchiveMark

	UUID       RoleBindingUUID `json:"uuid"` // PK
	TenantUUID TenantUUID      `json:"tenant_uuid"`
	Version    string          `json:"resource_version"`

	Description string `json:"description"`

	ValidTill  int64 `json:"valid_till"` // if ==0 => valid forever
	RequireMFA bool  `json:"require_mfa"`

	Users           []UserUUID           `json:"users"`
	Groups          []GroupUUID          `json:"groups"`
	ServiceAccounts []ServiceAccountUUID `json:"service_accounts"`
	Members         []MemberNotation     `json:"members"`

	AnyProject bool          `json:"any_project"`
	Projects   []ProjectUUID `json:"projects"`

	Roles []BoundRole `json:"roles"`

	Origin consts.ObjectOrigin `json:"origin"`

	Extensions map[consts.ObjectOrigin]*Extension `json:"-"`
}

type BoundRole struct {
	Name    RoleName               `json:"name"`
	Options map[string]interface{} `json:"options"`
}

func (r *RoleBinding) ObjType() string {
	return RoleBindingType
}

func (r *RoleBinding) ObjId() string {
	return r.UUID
}

// FixMembers remove from members invalid links, if some removed, returns true
func (r *RoleBinding) FixMembers() bool {
	return FixMembers(&r.Members, r.Users, r.Groups, r.ServiceAccounts)
}
