package model

type RoleResolver interface {
	IsUserSharedWith(TenantUUID) (bool, error)
	IsServiceAccountSharedWith(TenantUUID) (bool, error)

	CheckUserForProjectScopedRole(UserUUID, RoleName, TenantUUID, ProjectUUID) (bool, RoleBindingParams, error)
	CheckUserForTenantScopedRole(UserUUID, RoleName, TenantUUID) (bool, RoleBindingParams, error)
	CheckServiceAccountForProjectScopedRole(ServiceAccountUUID, RoleName, TenantUUID, ProjectUUID) (bool, RoleBindingParams, error)
	CheckServiceAccountForTenantScopedRole(ServiceAccountUUID, RoleName, TenantUUID) (bool, RoleBindingParams, error)

	FindSubjectsWithProjectScopedRole(RoleName, TenantUUID, ProjectUUID) ([]UserUUID, []ServiceAccountUUID, error)
	FindSubjectsWithTenantScopedRole(RoleName, TenantUUID) ([]UserUUID, []ServiceAccountUUID, error)

	CheckGroupForRole(GroupUUID, RoleName) (bool, error)
}

type RoleBindingParams struct {
	ValidTill  int64                  `json:"valid_till"`
	RequireMFA bool                   `json:"require_mfa"`
	Options    map[string]interface{} `json:"options"`
	// TODO approvals
	// TODO pendings
}
