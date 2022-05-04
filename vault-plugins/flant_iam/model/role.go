package model

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const RoleType = "role" // also, memdb schema name

type RoleScope string

const (
	RoleScopeTenant  RoleScope = "tenant"
	RoleScopeProject RoleScope = "project"
)

type Role struct {
	memdb.ArchiveMark

	Name              RoleName  `json:"name"`
	Scope             RoleScope `json:"scope"`
	TenantIsOptional  bool      `json:"tenant_is_optional"`
	ProjectIsOptional bool      `json:"project_is_optional"`

	Description   string `json:"description"`
	OptionsSchema string `json:"options_schema"`

	EnrichingExtensions []string `json:"enriching_extensions"`

	RequireOneOfFeatureFlags []FeatureFlagName `json:"require_one_of_feature_flags"`
	IncludedRoles            []IncludedRole    `json:"included_roles"`
	// FIXME add version?
}

type IncludedRole struct {
	Name            RoleName `json:"name"`
	OptionsTemplate string   `json:"options_template"`
}

func (r *Role) ObjType() string {
	return RoleType
}

func (r *Role) ObjId() string {
	return r.Name
}

// ValidateScope checks combinations of Scope, TenantIsOptional & ProjectIsOptional
// allowed combinations:
// scope: tenant tenant_is_optional: false (default)
// scope: tenant tenant_is_optional: true
// scope: project tenant_is_optional: false (default) project_is_optional: false (default)
// scope: project tenant_is_optional: false (default) project_is_optional: true
// scope: project tenant_is_optional: true project_is_optional: true
func (r *Role) ValidateScope() error {
	switch r.Scope {
	case RoleScopeTenant:
		if r.ProjectIsOptional {
			return fmt.Errorf("project_is_optional=true is prohibited for tenant scoped role")
		}
	case RoleScopeProject:
		if r.TenantIsOptional && !r.ProjectIsOptional {
			return fmt.Errorf("project_is_optional=false is prohibited for project scoped role, if passed tenant_id_optional=true")
		}
	default:
		return fmt.Errorf("scope %s is unknown", r.Scope)
	}
	return nil
}
