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

type RoleInformer interface {
	Get(RoleName) (*Role, error)
	FindAllIncludingRoles(RoleName) (map[RoleName]struct{}, error)
}

type GroupInformer interface {
	FindAllParentGroupsForUserUUID(TenantUUID, UserUUID) (map[GroupUUID]struct{}, error)
	FindAllParentGroupsForServiceAccountUUID(TenantUUID, ServiceAccountUUID) (map[GroupUUID]struct{}, error)
}

type RoleBindingsInformer interface {
	FindDirectRoleBindingsForTenantUser(TenantUUID, UserUUID) (map[RoleBindingUUID]*RoleBinding, error)
	FindDirectRoleBindingsForTenantServiceAccount(TenantUUID, ServiceAccountUUID) (map[RoleBindingUUID]*RoleBinding, error)
	FindDirectRoleBindingsForTenantGroups(TenantUUID, ...GroupUUID) (map[RoleBindingUUID]*RoleBinding, error)
	FindDirectRoleBindingsForTenantProject(TenantUUID, ProjectUUID) (map[RoleBindingUUID]struct{}, error)
	FindDirectRoleBindingsForRoles(TenantUUID, ...RoleName) (map[RoleBindingUUID]struct{}, error)
}

type roleResolver struct {
	ri  RoleInformer
	gi  GroupInformer
	rbi RoleBindingsInformer
}

var emptyRoleBindingParams = RoleBindingParams{}

func (r *roleResolver) IsUserSharedWith(uuid TenantUUID) (bool, error) {
	panic("implement me")
}

func (r *roleResolver) IsServiceAccountSharedWith(uuid TenantUUID) (bool, error) {
	panic("implement me")
}

func (r *roleResolver) CheckUserForProjectScopedRole(userUUID UserUUID, roleName RoleName, tenantUUID TenantUUID,
	projectUUID ProjectUUID) (bool, RoleBindingParams, error) {
	// TODO refactor it, after other 3 funcs will be designed
	roleNames, err := r.ri.FindAllIncludingRoles(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	groups, err := r.gi.FindAllParentGroupsForUserUUID(tenantUUID, userUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	fmt.Printf("roles : %#v\n", roleNames)
	fmt.Printf("groups : %#v\n", groups)
	userRBs, err := r.rbi.FindDirectRoleBindingsForTenantUser(tenantUUID, userUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	groupsRBs, err := r.rbi.FindDirectRoleBindingsForTenantGroups(tenantUUID, stringSlice(groups)...)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleBindings := groupsRBs
	for uuid, rb := range userRBs {
		roleBindings[uuid] = rb
	}
	roleBindingsForRoles, err := r.rbi.FindDirectRoleBindingsForRoles(tenantUUID, stringSlice(roleNames)...)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleBindingsForProject, err := r.rbi.FindDirectRoleBindingsForTenantProject(tenantUUID, projectUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 || len(roleBindingsForProject) == 0 {
		return false, emptyRoleBindingParams, nil
	}
	roleBindingParams := emptyRoleBindingParams
	roleExists := false
	for _, roleBinding := range roleBindings {
		_, rbHasRole := roleBindingsForRoles[roleBinding.UUID]
		_, rbHasProject := roleBindingsForProject[roleBinding.UUID]
		if rbHasProject && rbHasRole {
			roleBindingParams = mergeRoleBindingParams(roleBindingParams, roleBinding, roleNames)
			roleExists = true
		}
	}
	return roleExists, roleBindingParams, nil
}

func mergeRoleBindingParams(origin RoleBindingParams, roleBinding *RoleBinding, targetRoles map[RoleName]struct{}) RoleBindingParams {
	// TODO if several BoundRoles are from targetRoles, how to choose the best, or how to merge their options?
	// TODO how to merge? origin and chosen BoundRole?
	// now just take first and take the longest between chosen BoundRole and origin
	for _, boundRole := range roleBinding.Roles {
		if _, target := targetRoles[boundRole.Name]; target {
			if roleBinding.ValidTill > origin.ValidTill {
				origin = RoleBindingParams{
					ValidTill:  roleBinding.ValidTill,
					RequireMFA: roleBinding.RequireMFA,
					Options:    boundRole.Options,
				}
				break
			}
		}
	}
	return origin
}

func stringSlice(uuidSet map[string]struct{}) []string {
	result := make([]string, len(uuidSet), 0)
	for uuid := range uuidSet {
		result = append(result, uuid)
	}
	return result
}

func (r *roleResolver) CheckUserForTenantScopedRole(userUUID UserUUID, roleName RoleName,
	tenantUUID TenantUUID) (bool, RoleBindingParams, error) {
	role, err := r.ri.Get(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	if role.Scope == RoleScopeProject {
		return false, emptyRoleBindingParams, ErrBadProjectScopeRole
	}

	panic("implement me")
}

func (r *roleResolver) CheckServiceAccountForProjectScopedRole(uuid ServiceAccountUUID, name RoleName, uuid2 TenantUUID, uuid3 ProjectUUID) (bool, RoleBindingParams, error) {
	panic("implement me")
}

func (r *roleResolver) CheckServiceAccountForTenantScopedRole(uuid ServiceAccountUUID, name RoleName, uuid2 TenantUUID) (bool, RoleBindingParams, error) {
	panic("implement me")
}

func (r *roleResolver) FindSubjectsWithProjectScopedRole(name RoleName, uuid TenantUUID, uuid2 ProjectUUID) ([]UserUUID, []ServiceAccountUUID, error) {
	panic("implement me")
}

func (r *roleResolver) FindSubjectsWithTenantScopedRole(name RoleName, uuid TenantUUID) ([]UserUUID, []ServiceAccountUUID, error) {
	panic("implement me")
}

func (r *roleResolver) CheckGroupForRole(uuid GroupUUID, name RoleName) (bool, error) {
	panic("implement me")
}

func NewRoleResolver() (RoleResolver, error) {
	return &roleResolver{}, nil
}
