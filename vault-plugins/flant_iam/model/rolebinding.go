package model

const RoleBindingType = "role_binding" // also, memdb schema name

type RoleBinding struct {
	UUID       RoleBindingUUID `json:"uuid"` // PK
	TenantUUID TenantUUID      `json:"tenant_uuid"`
	Version    string          `json:"resource_version"`

	Identifier string `json:"identifier"`

	ValidTill  int64 `json:"valid_till"`
	RequireMFA bool  `json:"require_mfa"`

	Users           []UserUUID           `json:"-"`
	Groups          []GroupUUID          `json:"-"`
	ServiceAccounts []ServiceAccountUUID `json:"-"`
	Members         []MemberNotation     `json:"members"`

	AnyProject bool          `json:"any_project"`
	Projects   []ProjectUUID `json:"projects"`

	Roles []BoundRole `json:"roles"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"-"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

type BoundRole struct {
	Name    RoleName               `json:"name"`
	Options map[string]interface{} `json:"options"`
}

func (r *RoleBinding) IsDeleted() bool {
	return r.ArchivingTimestamp != 0
}

func (r *RoleBinding) ObjType() string {
	return RoleBindingType
}

func (r *RoleBinding) ObjId() string {
	return r.UUID
}
