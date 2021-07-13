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

func (r *roleResolver) IsUserSharedWith(TenantUUID) (bool, error) {
	panic("implement me")
}

func (r *roleResolver) IsServiceAccountSharedWith(TenantUUID) (bool, error) {
	panic("implement me")
}

func (r *roleResolver) collectAllRolesAndRoleBindings(tenantUUID TenantUUID,
	roleName RoleName) (map[RoleName]struct{}, map[RoleBindingUUID]struct{}, error) {
	roleNames, err := r.ri.FindAllIncludingRoles(roleName)
	if err != nil {
		return nil, nil, err
	}
	roleNames[roleName] = struct{}{}
	roleBindings, err := r.rbi.FindDirectRoleBindingsForRoles(tenantUUID, stringSlice(roleNames)...)
	if err != nil {
		return nil, nil, err
	}
	return roleNames, roleBindings, nil
}

func (r *roleResolver) collectAllRoleBindingsForUser(tenantUUID TenantUUID,
	userUUID UserUUID) (map[RoleBindingUUID]*RoleBinding, error) {
	groups, err := r.gi.FindAllParentGroupsForUserUUID(tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}
	fmt.Printf("groups : %#v\n", groups)
	userRBs, err := r.rbi.FindDirectRoleBindingsForTenantUser(tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}
	groupsRBs, err := r.rbi.FindDirectRoleBindingsForTenantGroups(tenantUUID, stringSlice(groups)...)
	if err != nil {
		return nil, err
	}
	roleBindings := groupsRBs
	for uuid, rb := range userRBs {
		roleBindings[uuid] = rb
	}
	return roleBindings, nil
}

func (r *roleResolver) collectAllRoleBindingsForServiceAccount(tenantUUID TenantUUID,
	serviceAccountUUID ServiceAccountUUID) (map[RoleBindingUUID]*RoleBinding, error) {
	groups, err := r.gi.FindAllParentGroupsForServiceAccountUUID(tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	fmt.Printf("groups : %#v\n", groups)
	serviceAccountRBs, err := r.rbi.FindDirectRoleBindingsForTenantServiceAccount(tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	groupsRBs, err := r.rbi.FindDirectRoleBindingsForTenantGroups(tenantUUID, stringSlice(groups)...)
	if err != nil {
		return nil, err
	}
	roleBindings := groupsRBs
	for uuid, rb := range serviceAccountRBs {
		roleBindings[uuid] = rb
	}
	return roleBindings, nil
}

func (r *roleResolver) CheckUserForProjectScopedRole(userUUID UserUUID, roleName RoleName, tenantUUID TenantUUID,
	projectUUID ProjectUUID) (bool, RoleBindingParams, error) {
	roleBindings, err := r.collectAllRoleBindingsForUser(tenantUUID, userUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleNames, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(tenantUUID, roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleBindingsForProject, err := r.rbi.FindDirectRoleBindingsForTenantProject(tenantUUID, projectUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, emptyRoleBindingParams, nil
	}
	roleBindingParams := emptyRoleBindingParams
	roleExists := false
	for _, roleBinding := range roleBindings {
		_, rbHasRole := roleBindingsForRoles[roleBinding.UUID]
		_, rbHasProject := roleBindingsForProject[roleBinding.UUID]
		if roleBinding.AnyProject {
			rbHasProject = true
		}
		if rbHasProject && rbHasRole {
			fmt.Printf("tatget rb UUID = %s\n", roleBinding.UUID)
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
	if len(uuidSet) == 0 {
		return nil
	}
	result := make([]string, 0, len(uuidSet))
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

	roleBindings, err := r.collectAllRoleBindingsForUser(tenantUUID, userUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleNames, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(tenantUUID, roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}

	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, emptyRoleBindingParams, nil
	}
	roleBindingParams := emptyRoleBindingParams
	roleExists := false
	for _, roleBinding := range roleBindings {
		if _, rbHasRole := roleBindingsForRoles[roleBinding.UUID]; rbHasRole {
			fmt.Printf("tatget rb UUID = %s\n", roleBinding.UUID)
			roleBindingParams = mergeRoleBindingParams(roleBindingParams, roleBinding, roleNames)
			roleExists = true
		}
	}
	return roleExists, roleBindingParams, nil
}

func (r *roleResolver) CheckServiceAccountForProjectScopedRole(serviceAccountUUID ServiceAccountUUID, roleName RoleName,
	tenantUUID TenantUUID, projectUUID ProjectUUID) (bool, RoleBindingParams, error) {
	roleBindings, err := r.collectAllRoleBindingsForServiceAccount(tenantUUID, serviceAccountUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleNames, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(tenantUUID, roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleBindingsForProject, err := r.rbi.FindDirectRoleBindingsForTenantProject(tenantUUID, projectUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, emptyRoleBindingParams, nil
	}
	roleBindingParams := emptyRoleBindingParams
	roleExists := false
	for _, roleBinding := range roleBindings {
		_, rbHasRole := roleBindingsForRoles[roleBinding.UUID]
		_, rbHasProject := roleBindingsForProject[roleBinding.UUID]
		if roleBinding.AnyProject {
			rbHasProject = true
		}
		if rbHasProject && rbHasRole {
			fmt.Printf("tatget rb UUID = %s\n", roleBinding.UUID)
			roleBindingParams = mergeRoleBindingParams(roleBindingParams, roleBinding, roleNames)
			roleExists = true
		}
	}
	return roleExists, roleBindingParams, nil
}

func (r *roleResolver) CheckServiceAccountForTenantScopedRole(serviceAccount ServiceAccountUUID, roleName RoleName,
	tenantUUID TenantUUID) (bool, RoleBindingParams, error) {
	role, err := r.ri.Get(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	if role.Scope == RoleScopeProject {
		return false, emptyRoleBindingParams, ErrBadProjectScopeRole
	}

	roleBindings, err := r.collectAllRoleBindingsForServiceAccount(tenantUUID, serviceAccount)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleNames, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(tenantUUID, roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}

	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, emptyRoleBindingParams, nil
	}
	roleBindingParams := emptyRoleBindingParams
	roleExists := false
	for _, roleBinding := range roleBindings {
		if _, rbHasRole := roleBindingsForRoles[roleBinding.UUID]; rbHasRole {
			fmt.Printf("tatget rb UUID = %s\n", roleBinding.UUID)
			roleBindingParams = mergeRoleBindingParams(roleBindingParams, roleBinding, roleNames)
			roleExists = true
		}
	}
	return roleExists, roleBindingParams, nil
}

func (r *roleResolver) FindSubjectsWithProjectScopedRole(RoleName, TenantUUID, ProjectUUID) ([]UserUUID, []ServiceAccountUUID, error) {
	panic("implement me")
}

func (r *roleResolver) FindSubjectsWithTenantScopedRole(RoleName, TenantUUID) ([]UserUUID, []ServiceAccountUUID, error) {
	panic("implement me")
}

func (r *roleResolver) CheckGroupForRole(GroupUUID, RoleName) (bool, error) {
	panic("implement me")
}

func NewRoleResolver(ri RoleInformer, gi GroupInformer, rbi RoleBindingsInformer) (RoleResolver, error) {
	return &roleResolver{
		ri:  ri,
		gi:  gi,
		rbi: rbi,
	}, nil
}
