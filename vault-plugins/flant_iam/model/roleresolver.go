package model

import "github.com/flant/negentropy/vault-plugins/shared/io"

type RoleResolver interface {
	IsUserSharedWithTenant(*User, TenantUUID) (bool, error)
	IsServiceAccountSharedWithTenant(*ServiceAccount, TenantUUID) (bool, error)

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
	FindAllSubjectsFor(TenantUUID, []UserUUID, []ServiceAccountUUID, []GroupUUID) (
		map[UserUUID]struct{}, map[ServiceAccountUUID]struct{}, error)
	FindAllParentGroupsForGroupUUID(TenantUUID, GroupUUID) (map[GroupUUID]struct{}, error)
	GetByID(GroupUUID) (*Group, error)
}

type RoleBindingsInformer interface {
	FindDirectRoleBindingsForTenantUser(TenantUUID, UserUUID) (map[RoleBindingUUID]*RoleBinding, error)
	FindDirectRoleBindingsForTenantServiceAccount(TenantUUID, ServiceAccountUUID) (map[RoleBindingUUID]*RoleBinding, error)
	FindDirectRoleBindingsForTenantGroups(TenantUUID, ...GroupUUID) (map[RoleBindingUUID]*RoleBinding, error)
	FindDirectRoleBindingsForTenantProject(TenantUUID, ProjectUUID) (map[RoleBindingUUID]*RoleBinding, error)
	FindDirectRoleBindingsForRoles(TenantUUID, ...RoleName) (map[RoleBindingUUID]*RoleBinding, error)
}

type SharingInformer interface {
	ListForDestinationTenant(tenantID TenantUUID) ([]*IdentitySharing, error)
}

type roleResolver struct {
	ri  RoleInformer
	gi  GroupInformer
	rbi RoleBindingsInformer
	si  SharingInformer
}

var emptyRoleBindingParams = RoleBindingParams{}

func (r *roleResolver) IsUserSharedWithTenant(user *User, destinationTenantUUID TenantUUID) (bool, error) {
	shares, err := r.si.ListForDestinationTenant(destinationTenantUUID)
	if err != nil {
		return false, err
	}
	sourceTenantGroups, err := r.gi.FindAllParentGroupsForUserUUID(user.TenantUUID, user.UUID)
	if err != nil {
		return false, err
	}
	for i := range shares {
		for _, sharedGroupUUID := range shares[i].Groups {
			if _, isShared := sourceTenantGroups[sharedGroupUUID]; isShared {
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *roleResolver) IsServiceAccountSharedWithTenant(serviceAccount *ServiceAccount, destinationTenantUUID TenantUUID) (
	bool, error) {
	shares, err := r.si.ListForDestinationTenant(destinationTenantUUID)
	if err != nil {
		return false, err
	}
	sourceTenantGroups, err := r.gi.FindAllParentGroupsForServiceAccountUUID(serviceAccount.TenantUUID, serviceAccount.UUID)
	if err != nil {
		return false, err
	}
	for i := range shares {
		for _, sharedGroupUUID := range shares[i].Groups {
			if _, isShared := sourceTenantGroups[sharedGroupUUID]; isShared {
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *roleResolver) collectAllRolesAndRoleBindings(tenantUUID TenantUUID,
	roleName RoleName) (map[RoleName]struct{}, map[RoleBindingUUID]*RoleBinding, error) {
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
			roleBindingParams = mergeRoleBindingParams(roleBindingParams, roleBinding, roleNames)
			roleExists = true
		}
	}
	return roleExists, roleBindingParams, nil
}

func (r *roleResolver) CheckServiceAccountForProjectScopedRole(serviceAccountUUID ServiceAccountUUID, roleName RoleName, tenantUUID TenantUUID, projectUUID ProjectUUID) (bool, RoleBindingParams, error) {
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
			roleBindingParams = mergeRoleBindingParams(roleBindingParams, roleBinding, roleNames)
			roleExists = true
		}
	}
	return roleExists, roleBindingParams, nil
}

func (r *roleResolver) FindSubjectsWithProjectScopedRole(roleName RoleName, tenantUUID TenantUUID,
	projectUUID ProjectUUID) ([]UserUUID, []ServiceAccountUUID, error) {
	_, roleBindings, err := r.collectAllRolesAndRoleBindings(tenantUUID, roleName)
	if err != nil {
		return nil, nil, err
	}
	if len(roleBindings) == 0 {
		return nil, nil, nil
	}
	roleBindingsForProject, err := r.rbi.FindDirectRoleBindingsForTenantProject(tenantUUID, projectUUID)
	if err != nil {
		return nil, nil, err
	}
	users := map[UserUUID]struct{}{}
	serviceAccounts := map[ServiceAccountUUID]struct{}{}
	groups := map[GroupUUID]struct{}{}
	for _, rb := range roleBindings {
		if _, hasProject := roleBindingsForProject[rb.UUID]; hasProject || rb.AnyProject {
			users = mergeUUIDs(users, rb.Users)
			serviceAccounts = mergeUUIDs(serviceAccounts, rb.ServiceAccounts)
			groups = mergeUUIDs(groups, rb.Groups)
		}
	}
	users, serviceAccounts, err = r.gi.FindAllSubjectsFor(tenantUUID,
		stringSlice(users), stringSlice(serviceAccounts), stringSlice(groups))
	if err != nil {
		return nil, nil, err
	}
	return stringSlice(users), stringSlice(serviceAccounts), nil
}

func mergeUUIDs(originUUIDs map[string]struct{}, extraUUIDs []string) map[string]struct{} {
	for i := range extraUUIDs {
		originUUIDs[extraUUIDs[i]] = struct{}{}
	}
	return originUUIDs
}

func (r *roleResolver) FindSubjectsWithTenantScopedRole(roleName RoleName, tenantUUID TenantUUID) ([]UserUUID, []ServiceAccountUUID, error) {
	role, err := r.ri.Get(roleName)
	if err != nil {
		return nil, nil, err
	}
	if role.Scope == RoleScopeProject {
		return nil, nil, ErrBadProjectScopeRole
	}
	_, roleBindings, err := r.collectAllRolesAndRoleBindings(tenantUUID, roleName)
	if err != nil {
		return nil, nil, err
	}
	if len(roleBindings) == 0 {
		return nil, nil, nil
	}
	users := map[UserUUID]struct{}{}
	serviceAccounts := map[ServiceAccountUUID]struct{}{}
	groups := map[GroupUUID]struct{}{}
	for _, rb := range roleBindings {
		users = mergeUUIDs(users, rb.Users)
		serviceAccounts = mergeUUIDs(serviceAccounts, rb.ServiceAccounts)
		groups = mergeUUIDs(groups, rb.Groups)
	}
	users, serviceAccounts, err = r.gi.FindAllSubjectsFor(tenantUUID,
		stringSlice(users), stringSlice(serviceAccounts), stringSlice(groups))
	if err != nil {
		return nil, nil, err
	}
	return stringSlice(users), stringSlice(serviceAccounts), nil
}

func (r *roleResolver) CheckGroupForRole(groupUUID GroupUUID, roleName RoleName) (bool, error) {
	group, err := r.gi.GetByID(groupUUID)
	if err != nil {
		return false, err
	}
	tenantUUID := group.TenantUUID
	groupUUIDs, err := r.gi.FindAllParentGroupsForGroupUUID(tenantUUID, groupUUID)
	if err != nil {
		return false, err
	}
	roleBindingsForGroup, err := r.rbi.FindDirectRoleBindingsForTenantGroups(tenantUUID, stringSlice(groupUUIDs)...)
	if err != nil {
		return false, err
	}
	_, roleBindingsForRole, err := r.collectAllRolesAndRoleBindings(tenantUUID, roleName)
	if err != nil {
		return false, err
	}
	for rbUUID := range roleBindingsForRole {
		if _, found := roleBindingsForGroup[rbUUID]; found {
			return true, nil
		}
	}
	return false, err
}

func NewRoleResolver(tx *io.MemoryStoreTxn) RoleResolver {
	return &roleResolver{
		ri:  NewRoleRepository(tx),
		gi:  NewGroupRepository(tx),
		rbi: NewRoleBindingRepository(tx),
		si:  NewIdentitySharingRepository(tx),
	}
}
