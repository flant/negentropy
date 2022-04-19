package usecase

import (
	"fmt"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleResolver interface {
	IsUserSharedWithTenant(model.UserUUID, model.TenantUUID) (bool, error)
	IsServiceAccountSharedWithTenant(model.ServiceAccountUUID, model.TenantUUID) (bool, error)

	CheckUserForProjectScopedRole(model.UserUUID, model.RoleName, model.ProjectUUID) (bool, []EffectiveRole, error)
	CheckUserForTenantScopedRole(model.UserUUID, model.RoleName, model.TenantUUID) (bool, []EffectiveRole, error)
	CheckServiceAccountForProjectScopedRole(model.ServiceAccountUUID, model.RoleName, model.ProjectUUID) (bool, []EffectiveRole, error)
	CheckServiceAccountForTenantScopedRole(model.ServiceAccountUUID, model.RoleName, model.TenantUUID) (bool, []EffectiveRole, error)

	FindMembersWithProjectScopedRole(model.RoleName, model.TenantUUID, model.ProjectUUID) ([]model.UserUUID, []model.ServiceAccountUUID, error)
	FindMembersWithTenantScopedRole(model.RoleName, model.TenantUUID) ([]model.UserUUID, []model.ServiceAccountUUID, error)

	CheckGroupForRole(model.GroupUUID, model.RoleName) (bool, error)
}

type EffectiveRole struct {
	RoleName        model.RoleName         `json:"rolename"`
	RoleBindingUUID model.RoleBindingUUID  `json:"rolebinding_uuid"`
	TenantUUID      model.TenantUUID       `json:"tenant_uuid"`
	ValidTill       int64                  `json:"valid_till"`
	RequireMFA      bool                   `json:"require_mfa"`
	AnyProject      bool                   `json:"any_project"`
	Projects        []model.ProjectUUID    `json:"projects"`
	NeedApprovals   int64                  `json:"need_approvals"`
	Options         map[string]interface{} `json:"options"`
}

type RoleInformer interface {
	GetByID(model.RoleName) (*model.Role, error)
	FindAllAncestorsRoles(model.RoleName) (map[model.RoleName]repo.RoleChain, error)
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

type ApprovalInformer interface {
	RoleBindingApprovalCount(rolebindingUUID model.RoleBindingApprovalUUID) int64
}

type roleResolver struct {
	roleInformer         RoleInformer
	groupInformer        GroupInformer
	roleBindingsInformer RoleBindingsInformer
	sharingInformer      SharingInformer
	approvalInformer     ApprovalInformer
}

func (r *roleResolver) IsUserSharedWithTenant(userUUID model.UserUUID, destinationTenantUUID model.TenantUUID) (bool, error) {
	shares, err := r.sharingInformer.ListForDestinationTenant(destinationTenantUUID)
	if err != nil {
		return false, err
	}
	sourceTenantGroups, err := r.groupInformer.FindAllParentGroupsForUserUUID(userUUID)
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

func (r *roleResolver) IsServiceAccountSharedWithTenant(serviceAccountUUID model.ServiceAccountUUID, destinationTenantUUID model.TenantUUID) (
	bool, error) {
	shares, err := r.sharingInformer.ListForDestinationTenant(destinationTenantUUID)
	if err != nil {
		return false, err
	}
	sourceTenantGroups, err := r.groupInformer.FindAllParentGroupsForServiceAccountUUID(serviceAccountUUID)
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

func (r *roleResolver) collectAllRolesAndRoleBindings(roleName model.RoleName) (map[model.RoleName]repo.RoleChain,
	map[model.RoleBindingUUID]*model.RoleBinding, error) {
	roles, err := r.roleInformer.FindAllAncestorsRoles(roleName)
	if err != nil {
		return nil, nil, err
	}

	roleBindings, err := r.roleBindingsInformer.FindDirectRoleBindingsForRoles(roleNames(roles)...)
	if err != nil {
		return nil, nil, err
	}
	return roles, roleBindings, nil
}

func (r *roleResolver) collectAllRoleBindingsForUser(
	userUUID model.UserUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	groups, err := r.groupInformer.FindAllParentGroupsForUserUUID(userUUID)
	if err != nil {
		return nil, err
	}
	userRBs, err := r.roleBindingsInformer.FindDirectRoleBindingsForUser(userUUID)
	if err != nil {
		return nil, err
	}
	groupsRBs, err := r.roleBindingsInformer.FindDirectRoleBindingsForGroups(stringSlice(groups)...)
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
	groups, err := r.groupInformer.FindAllParentGroupsForServiceAccountUUID(serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	serviceAccountRBs, err := r.roleBindingsInformer.FindDirectRoleBindingsForServiceAccount(serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	groupsRBs, err := r.roleBindingsInformer.FindDirectRoleBindingsForGroups(stringSlice(groups)...)
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
	projectUUID model.ProjectUUID) (bool, []EffectiveRole, error) {
	role, err := r.roleInformer.GetByID(roleName)
	if err != nil {
		return false, nil, err
	}
	if role.Scope != model.RoleScopeProject {
		return false, nil, consts.ErrBadScopeRole
	}
	roleBindings, err := r.collectAllRoleBindingsForUser(userUUID)
	fmt.Printf("roleBindings, err := r.collectAllRoleBindingsForUser(userUUID): %#v\n", roleBindings) // TODO REMOVE !!!!

	if err != nil {
		return false, nil, err
	}
	roles, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName)
	fmt.Printf("roles, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName): %#v %#v\n", roles, roleBindingsForRoles) // TODO REMOVE !!!!

	if err != nil {
		return false, nil, err
	}
	roleBindingsForProject, err := r.roleBindingsInformer.FindDirectRoleBindingsForProject(projectUUID)
	fmt.Printf("roleBindingsForProject, err := r.roleBindingsInformer.FindDirectRoleBindingsForProject(projectUUID): %#v\n", roleBindingsForProject) // TODO REMOVE !!!!
	if err != nil {
		return false, nil, err
	}
	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, nil, nil
	}
	effectiveRoles := []EffectiveRole{}
	roleExists := false
	for _, roleBinding := range roleBindings {
		_, rbHasRole := roleBindingsForRoles[roleBinding.UUID]
		_, rbHasProject := roleBindingsForProject[roleBinding.UUID]
		if roleBinding.AnyProject {
			rbHasProject = true
		}
		if rbHasProject && rbHasRole {
			effectiveRoles = r.mergeEffectiveRoles(effectiveRoles, roleBinding, roles, roleName)
			roleExists = true
		}
	}
	return roleExists, effectiveRoles, nil
}

func (r *roleResolver) mergeEffectiveRoles(originEffectiveRoles []EffectiveRole, roleBinding *model.RoleBinding,
	targetRoles map[model.RoleName]repo.RoleChain, masterRole model.RoleName) []EffectiveRole {
	result := originEffectiveRoles
	for _, boundRole := range roleBinding.Roles {
		if roleOptionsTemplatesChain, target := targetRoles[boundRole.Name]; target {
			if roleBinding.ValidTill == 0 || roleBinding.ValidTill > time.Now().Unix() {
				newEffectiveRole := EffectiveRole{
					RoleName:        masterRole,
					RoleBindingUUID: roleBinding.UUID,
					TenantUUID:      roleBinding.TenantUUID,
					ValidTill:       roleBinding.ValidTill,
					RequireMFA:      roleBinding.RequireMFA,
					AnyProject:      roleBinding.AnyProject,
					Projects:        roleBinding.Projects,
					NeedApprovals:   r.approvalInformer.RoleBindingApprovalCount(roleBinding.UUID),
					Options:         applyRoleOptionsTemplatesChain(boundRole.Options, roleOptionsTemplatesChain),
				}
				result = append(result, newEffectiveRole)
				break
			}
		}
	}
	return result
}

func applyRoleOptionsTemplatesChain(options map[string]interface{}, chain repo.RoleChain) map[string]interface{} {
	for _, template := range chain.OptionsTemplates {
		options = applyOptionTemplate(options, template)
	}
	return options
}

func applyOptionTemplate(options map[string]interface{}, template string) map[string]interface{} {
	// TODO implement this!
	return options
}

func (r *roleResolver) CheckUserForTenantScopedRole(userUUID model.UserUUID, roleName model.RoleName,
	tenantUUID model.TenantUUID) (bool, []EffectiveRole, error) {
	role, err := r.roleInformer.GetByID(roleName)
	if err != nil {
		return false, nil, err
	}
	if role.Scope != model.RoleScopeTenant {
		return false, nil, consts.ErrBadScopeRole
	}

	roleBindings, err := r.collectAllRoleBindingsForUser(userUUID)
	if err != nil {
		return false, nil, err
	}
	roles, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return false, nil, err
	}

	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, nil, nil
	}
	effectiveRoles := []EffectiveRole{}
	roleExists := false
	for _, roleBinding := range roleBindings {
		if roleBinding.TenantUUID != tenantUUID {
			continue
		}
		if _, rbHasRole := roleBindingsForRoles[roleBinding.UUID]; rbHasRole {
			effectiveRoles = r.mergeEffectiveRoles(effectiveRoles, roleBinding, roles, roleName)
			roleExists = true
		}
	}
	return roleExists, effectiveRoles, nil
}

func (r *roleResolver) CheckServiceAccountForProjectScopedRole(serviceAccountUUID model.ServiceAccountUUID,
	roleName model.RoleName, projectUUID model.ProjectUUID) (bool, []EffectiveRole, error) {
	role, err := r.roleInformer.GetByID(roleName)
	if err != nil {
		return false, nil, err
	}
	if role.Scope != model.RoleScopeProject {
		return false, nil, consts.ErrBadScopeRole
	}
	roleBindings, err := r.collectAllRoleBindingsForServiceAccount(serviceAccountUUID)
	if err != nil {
		return false, nil, err
	}
	roles, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return false, nil, err
	}
	roleBindingsForProject, err := r.roleBindingsInformer.FindDirectRoleBindingsForProject(projectUUID)
	if err != nil {
		return false, nil, err
	}
	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, nil, nil
	}
	effectiveRoles := []EffectiveRole{}
	roleExists := false
	for _, roleBinding := range roleBindings {
		_, rbHasRole := roleBindingsForRoles[roleBinding.UUID]
		_, rbHasProject := roleBindingsForProject[roleBinding.UUID]
		if roleBinding.AnyProject {
			rbHasProject = true
		}
		if rbHasProject && rbHasRole {
			effectiveRoles = r.mergeEffectiveRoles(effectiveRoles, roleBinding, roles, roleName)
			roleExists = true
		}
	}
	return roleExists, effectiveRoles, nil
}

func (r *roleResolver) CheckServiceAccountForTenantScopedRole(serviceAccount model.ServiceAccountUUID, roleName model.RoleName,
	tenantUUID model.TenantUUID) (bool, []EffectiveRole, error) {
	role, err := r.roleInformer.GetByID(roleName)
	if err != nil {
		return false, nil, err
	}
	if role.Scope != model.RoleScopeTenant {
		return false, nil, consts.ErrBadScopeRole
	}

	roleBindings, err := r.collectAllRoleBindingsForServiceAccount(serviceAccount)
	if err != nil {
		return false, nil, err
	}
	roles, roleBindingsForRoles, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return false, nil, err
	}

	if len(roleBindings) == 0 || len(roleBindingsForRoles) == 0 {
		return false, nil, nil
	}
	effectiveRoles := []EffectiveRole{}
	roleExists := false
	for _, roleBinding := range roleBindings {
		if roleBinding.TenantUUID != tenantUUID {
			continue
		}
		if _, rbHasRole := roleBindingsForRoles[roleBinding.UUID]; rbHasRole {
			effectiveRoles = r.mergeEffectiveRoles(effectiveRoles, roleBinding, roles, roleName)
			roleExists = true
		}
	}
	return roleExists, effectiveRoles, nil
}

func (r *roleResolver) FindMembersWithProjectScopedRole(roleName model.RoleName, tenantUUID model.TenantUUID,
	projectUUID model.ProjectUUID) ([]model.UserUUID, []model.ServiceAccountUUID, error) {
	_, roleBindings, err := r.collectAllRolesAndRoleBindings(roleName)
	if err != nil {
		return nil, nil, err
	}
	if len(roleBindings) == 0 {
		return nil, nil, nil
	}
	roleBindingsForProject, err := r.roleBindingsInformer.FindDirectRoleBindingsForProject(projectUUID)
	if err != nil {
		return nil, nil, err
	}
	users := map[model.UserUUID]struct{}{}
	serviceAccounts := map[model.ServiceAccountUUID]struct{}{}
	groups := map[model.GroupUUID]struct{}{}
	for _, rb := range roleBindings {
		if _, hasProject := roleBindingsForProject[rb.UUID]; hasProject || (rb.AnyProject && rb.TenantUUID == tenantUUID) {
			users = mergeUUIDs(users, rb.Users)
			serviceAccounts = mergeUUIDs(serviceAccounts, rb.ServiceAccounts)
			groups = mergeUUIDs(groups, rb.Groups)
		}
	}
	users, serviceAccounts, err = r.groupInformer.FindAllMembersFor(stringSlice(users), stringSlice(serviceAccounts),
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
	role, err := r.roleInformer.GetByID(roleName)
	if err != nil {
		return nil, nil, err
	}
	if role.Scope == model.RoleScopeProject {
		return nil, nil, consts.ErrBadScopeRole
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
	users, serviceAccounts, err = r.groupInformer.FindAllMembersFor(stringSlice(users),
		stringSlice(serviceAccounts), stringSlice(groups))
	if err != nil {
		return nil, nil, err
	}
	return stringSlice(users), stringSlice(serviceAccounts), nil
}

func (r *roleResolver) CheckGroupForRole(groupUUID model.GroupUUID, roleName model.RoleName) (bool, error) {
	groupUUIDs, err := r.groupInformer.FindAllParentGroupsForGroupUUID(groupUUID)
	if err != nil {
		return false, err
	}
	roleBindingsForGroup, err := r.roleBindingsInformer.FindDirectRoleBindingsForGroups(stringSlice(groupUUIDs)...)
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
		roleInformer:         repo.NewRoleRepository(tx),
		groupInformer:        repo.NewGroupRepository(tx),
		roleBindingsInformer: repo.NewRoleBindingRepository(tx),
		sharingInformer:      repo.NewIdentitySharingRepository(tx),
		approvalInformer:     repo.NewRoleBindingApprovalRepository(tx),
	}
}
