package model

const RoleType = "role" // also, memdb schema name

type RoleScope string

const (
	RoleScopeTenant  RoleScope = "tenant"
	RoleScopeProject RoleScope = "project"
)

type Role struct {
	Name  RoleName  `json:"name"`
	Scope RoleScope `json:"scope"`

	Description   string `json:"description"`
	OptionsSchema string `json:"options_schema"`

	RequireOneOfFeatureFlags []FeatureFlagName `json:"require_one_of_feature_flags"`
	IncludedRoles            []IncludedRole    `json:"included_roles"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
	// FIXME add version?
}

type IncludedRole struct {
	Name            RoleName `json:"name"`
	OptionsTemplate string   `json:"options_template"`
}

func (r *Role) IsDeleted() bool {
	return r.ArchivingTimestamp != 0
}

func (r *Role) ObjType() string {
	return RoleType
}

func (r *Role) ObjId() string {
	return r.Name
}
