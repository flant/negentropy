package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleResolver interface {
	IsUserSharedWithTenant(*model.User, model.TenantUUID) (bool, error)
	IsServiceAccountSharedWithTenant(*model.ServiceAccount, model.TenantUUID) (bool, error)

	CheckUserForProjectScopedRole(model.UserUUID, model.RoleName, model.ProjectUUID) (bool, RoleBindingParams, error)
	CheckUserForTenantScopedRole(model.UserUUID, model.RoleName, model.TenantUUID) (bool, RoleBindingParams, error)
	CheckServiceAccountForProjectScopedRole(model.ServiceAccountUUID, model.RoleName, model.ProjectUUID) (bool, RoleBindingParams, error)
	CheckServiceAccountForTenantScopedRole(model.ServiceAccountUUID, model.RoleName, model.TenantUUID) (bool, RoleBindingParams, error)

	FindMembersWithProjectScopedRole(model.RoleName, model.ProjectUUID) ([]model.UserUUID, []model.ServiceAccountUUID, error)
	FindMembersWithTenantScopedRole(model.RoleName, model.TenantUUID) ([]model.UserUUID, []model.ServiceAccountUUID, error)

	CheckGroupForRole(model.GroupUUID, model.RoleName) (bool, error)
}

type RoleBindingParams struct {
	ValidTill  int64                  `json:"valid_till"`
	RequireMFA bool                   `json:"require_mfa"`
	Options    map[string]interface{} `json:"options"`
	// TODO approvals
	// TODO pendings
}

type RoleInformer interface {
	GetByID(model.RoleName) (*model.Role, error)
	FindAllIncludingRoles(model.RoleName) (map[model.RoleName]struct{}, error)
}

type GroupInformer interface {
	FindAllParentGroupsForUserUUID(model.UserUUID) (map[model.GroupUUID]struct{}, error)
	FindAllParentGroupsForServiceAccountUUID(model.ServiceAccountUUID) (map[model.GroupUUID]struct{}, error)
	FindAllParentGroupsForGroupUUID(model.GroupUUID) (map[model.GroupUUID]struct{}, error)
	FindAllMembersFor([]model.UserUUID, []model.ServiceAccountUUID, []model.GroupUUID) (
		map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, error)
	GetByID(model.GroupUUID) (*model.Group, error)
}

type RoleBindingsInformer interface {
	FindDirectRoleBindingsForUser(model.UserUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error)
	FindDirectRoleBindingsForServiceAccount(model.ServiceAccountUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error)
	FindDirectRoleBindingsForGroups(...model.GroupUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error)
	FindDirectRoleBindingsForProject(model.ProjectUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error)
	FindDirectRoleBindingsForRoles(...model.RoleName) (map[model.RoleBindingUUID]*model.RoleBinding, error)
}

type SharingInformer interface {
	ListForDestinationTenant(tenantID model.TenantUUID) ([]*model.IdentitySharing, error)
}

type roleResolver struct {
	ri  RoleInformer
	gi  GroupInformer
	rbi RoleBindingsInformer
	si  SharingInformer
}

var emptyRoleBindingParams = RoleBindingParams{}

func (r *roleResolver) IsUserSharedWithTenant(user *model.User, destinationTenantUUID model.TenantUUID) (bool, error) {
	shares, err := r.si.ListForDestinationTenant(destinationTenantUUID)
	if err != nil {
		return false, err
	}
	sourceTenantGroups, err := r.gi.FindAllParentGroupsForUserUUID(user.UUID)
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

func (r *roleResolver) IsServiceAccountSharedWithTenant(serviceAccount *model.ServiceAccount, destinationTenantUUID model.TenantUUID) (
	bool, error) {
	shares, err := r.si.ListForDestinationTenant(destinationTenantUUID)
	if err != nil {
		return false, err
	}
	sourceTenantGroups, err := r.gi.FindAllParentGroupsForServiceAccountUUID(serviceAccount.UUID)
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

func (r *roleResolver) collectAllRolesAndRoleBindings(roleName model.RoleName) (map[model.RoleName]struct{},
	map[model.RoleBindingUUID]*model.RoleBinding, error) {
	roleNames, err := r.ri.FindAllIncludingRoles(roleName)
	if err != nil {
		return nil, nil, err
	}
	roleNames[roleName] = struct{}{}
	roleBindings, err := r.rbi.FindDirectRoleBindingsForRoles(stringSlice(roleNames)...)
	if err != nil {
		return nil, nil, err
	}
	return roleNames, roleBindings, nil
}

func (r *roleResolver) collectAllRoleBindingsForUser(
	userUUID model.UserUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	groups, err := r.gi.FindAllParentGroupsForUserUUID(userUUID)
	if err != nil {
		return nil, err
	}
	userRBs, err := r.rbi.FindDirectRoleBindingsForUser(userUUID)
	if err != nil {
		return nil, err
	}
	groupsRBs, err := r.rbi.FindDirectRoleBindingsForGroups(stringSlice(groups)...)
	if err != nil {
		return nil, err
	}
	roleBindings := groupsRBs
	for uuid, rb := range userRBs {
		roleBindings[uuid] = rb
	}
	return roleBindings, nil
}

func (r *roleResolver) collectAllRoleBindingsForServiceAccount(
	serviceAccountUUID model.ServiceAccountUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	groups, err := r.gi.FindAllParentGroupsForServiceAccountUUID(serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	serviceAccountRBs, err := r.rbi.FindDirectRoleBindingsForServiceAccount(serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	groupsRBs, err := r.rbi.FindDirectRoleBindingsForGroups(stringSlice(groups)...)
	if err != nil {
		return nil, err
	}
	roleBindings := groupsRBs
	for uuid, rb := range serviceAccountRBs {
		roleBindings[uuid] = rb
	}
	return roleBindings, nil
}

func (r *roleResolver) CheckUserForProjectScopedRole(userUUID model.UserUUID, roleName model.RoleName,
	projectUUID model.ProjectUUID) (bool, RoleBindingParams, error) {
	roleBindings, err := r.collectAllRoleBindingsForUser(userUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleNames, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleBindingsForProject, err := r.rbi.FindDirectRoleBindingsForProject(projectUUID)
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

func mergeRoleBindingParams(origin RoleBindingParams, roleBinding *model.RoleBinding, targetRoles map[model.RoleName]struct{}) RoleBindingParams {
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

func (r *roleResolver) CheckUserForTenantScopedRole(userUUID model.UserUUID, roleName model.RoleName,
	tenantUUID model.TenantUUID) (bool, RoleBindingParams, error) {
	role, err := r.ri.GetByID(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	if role.Scope == model.RoleScopeProject {
		return false, emptyRoleBindingParams, model.ErrBadProjectScopeRole
	}

	roleBindings, err := r.collectAllRoleBindingsForUser(userUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleNames, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}

	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, emptyRoleBindingParams, nil
	}
	roleBindingParams := emptyRoleBindingParams
	roleExists := false
	for _, roleBinding := range roleBindings {
		if roleBinding.TenantUUID != tenantUUID {
			continue
		}
		if _, rbHasRole := roleBindingsForRoles[roleBinding.UUID]; rbHasRole {
			roleBindingParams = mergeRoleBindingParams(roleBindingParams, roleBinding, roleNames)
			roleExists = true
		}
	}
	return roleExists, roleBindingParams, nil
}

func (r *roleResolver) CheckServiceAccountForProjectScopedRole(serviceAccountUUID model.ServiceAccountUUID,
	roleName model.RoleName, projectUUID model.ProjectUUID) (bool, RoleBindingParams, error) {
	roleBindings, err := r.collectAllRoleBindingsForServiceAccount(serviceAccountUUID)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleNames, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleBindingsForProject, err := r.rbi.FindDirectRoleBindingsForProject(projectUUID)
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

func (r *roleResolver) CheckServiceAccountForTenantScopedRole(serviceAccount model.ServiceAccountUUID, roleName model.RoleName,
	tenantUUID model.TenantUUID) (bool, RoleBindingParams, error) {
	role, err := r.ri.GetByID(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	if role.Scope == model.RoleScopeProject {
		return false, emptyRoleBindingParams, model.ErrBadProjectScopeRole
	}

	roleBindings, err := r.collectAllRoleBindingsForServiceAccount(serviceAccount)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}
	roleNames, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return false, emptyRoleBindingParams, err
	}

	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, emptyRoleBindingParams, nil
	}
	roleBindingParams := emptyRoleBindingParams
	roleExists := false
	for _, roleBinding := range roleBindings {
		if roleBinding.TenantUUID != tenantUUID {
			continue
		}
		if _, rbHasRole := roleBindingsForRoles[roleBinding.UUID]; rbHasRole {
			roleBindingParams = mergeRoleBindingParams(roleBindingParams, roleBinding, roleNames)
			roleExists = true
		}
	}
	return roleExists, roleBindingParams, nil
}

func (r *roleResolver) FindMembersWithProjectScopedRole(roleName model.RoleName,
	projectUUID model.ProjectUUID) ([]model.UserUUID, []model.ServiceAccountUUID, error) {
	_, roleBindings, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return nil, nil, err
	}
	if len(roleBindings) == 0 {
		return nil, nil, nil
	}
	roleBindingsForProject, err := r.rbi.FindDirectRoleBindingsForProject(projectUUID)
	if err != nil {
		return nil, nil, err
	}
	users := map[model.UserUUID]struct{}{}
	serviceAccounts := map[model.ServiceAccountUUID]struct{}{}
	groups := map[model.GroupUUID]struct{}{}
	for _, rb := range roleBindings {
		if _, hasProject := roleBindingsForProject[rb.UUID]; hasProject || rb.AnyProject {
			users = mergeUUIDs(users, rb.Users)
			serviceAccounts = mergeUUIDs(serviceAccounts, rb.ServiceAccounts)
			groups = mergeUUIDs(groups, rb.Groups)
		}
	}
	users, serviceAccounts, err = r.gi.FindAllMembersFor(stringSlice(users), stringSlice(serviceAccounts),
		stringSlice(groups))
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

func (r *roleResolver) FindMembersWithTenantScopedRole(roleName model.RoleName, tenantUUID model.TenantUUID) ([]model.UserUUID, []model.ServiceAccountUUID, error) {
	role, err := r.ri.GetByID(roleName)
	if err != nil {
		return nil, nil, err
	}
	if role.Scope == model.RoleScopeProject {
		return nil, nil, model.ErrBadProjectScopeRole
	}
	_, roleBindings, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return nil, nil, err
	}
	if len(roleBindings) == 0 {
		return nil, nil, nil
	}
	users := map[model.UserUUID]struct{}{}
	serviceAccounts := map[model.ServiceAccountUUID]struct{}{}
	groups := map[model.GroupUUID]struct{}{}
	for _, rb := range roleBindings {
		if rb.TenantUUID != tenantUUID {
			continue
		}
		users = mergeUUIDs(users, rb.Users)
		serviceAccounts = mergeUUIDs(serviceAccounts, rb.ServiceAccounts)
		groups = mergeUUIDs(groups, rb.Groups)
	}
	users, serviceAccounts, err = r.gi.FindAllMembersFor(stringSlice(users),
		stringSlice(serviceAccounts), stringSlice(groups))
	if err != nil {
		return nil, nil, err
	}
	return stringSlice(users), stringSlice(serviceAccounts), nil
}

func (r *roleResolver) CheckGroupForRole(groupUUID model.GroupUUID, roleName model.RoleName) (bool, error) {
	groupUUIDs, err := r.gi.FindAllParentGroupsForGroupUUID(groupUUID)
	if err != nil {
		return false, err
	}
	roleBindingsForGroup, err := r.rbi.FindDirectRoleBindingsForGroups(stringSlice(groupUUIDs)...)
	if err != nil {
		return false, err
	}
	_, roleBindingsForRole, err := r.collectAllRolesAndRoleBindings(roleName)
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
		ri:  repo.NewRoleRepository(tx),
		gi:  repo.NewGroupRepository(tx),
		rbi: repo.NewRoleBindingRepository(tx),
		si:  repo.NewIdentitySharingRepository(tx),
	}
}
