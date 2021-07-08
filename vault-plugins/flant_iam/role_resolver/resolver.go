package role_resolver

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// FindUsersAndSAsAffectedByPossibleRoleAddingOnRoleBindingChange
func FindUsersAndSAsAffectedByPossibleRoleAddingOnRoleBindingChange(txn *io.MemoryStoreTxn, oldRoleBinding *model.RoleBinding, newRoleBinding *model.RoleBinding, roleOfConcern string) (map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, error) {
	// There is no way to add role by role binding deletion.
	if newRoleBinding == nil {
		return nil, nil, nil
	}

	// Check if role might be granted for newBindingRole members.
	newRoleBindingGrantsRole, err := CheckRoleBindingGrantsRole(txn, newRoleBinding, roleOfConcern)
	if err != nil {
		return nil, nil, err
	}
	// Return nothing if role is not granted in updated RoleBinding.
	if !newRoleBindingGrantsRole {
		return nil, nil, nil
	}

	// Check if role might be granted for oldBindingRole members.
	oldRoleBindingGrantsRole, err := CheckRoleBindingGrantsRole(txn, oldRoleBinding, roleOfConcern)
	if err != nil {
		return nil, nil, err
	}

	groupRepo := model.NewGroupRepository(txn)
	// Return all members of the new RoleBinding if role was not granted
	// by the old RoleBinding and become granted by the new RoleBinding.
	if !oldRoleBindingGrantsRole && newRoleBindingGrantsRole {
		return groupRepo.FindAllSubjectsFor(newRoleBinding.TenantUUID, newRoleBinding.Users, newRoleBinding.ServiceAccounts, newRoleBinding.Groups)
	}

	// Return added subjects if role is not changed.
	// Calculate added users.
	addedUsers, addedSAs, addedGroups := calculateAddedSubjectsForRoleBinding(oldRoleBinding, newRoleBinding)

	if len(addedUsers) == 0 && len(addedSAs) == 0 && len(addedGroups) == 0 {
		return nil, nil, nil
	}

	return groupRepo.FindAllSubjectsFor(newRoleBinding.TenantUUID, arrayFromMap(addedUsers), arrayFromMap(addedSAs), arrayFromMap(addedGroups))
}

// FindUsersAndSAsAffectedByPossibleRoleAddingOnGroupChange
// Return empty if group is added or deleted.
// Return empty if group is modified without adding new users.
// Return empty if group and parent groups have no role binding with the specified role.
// Return newly added users if group is modified and there are role bindings
// for group or for parent group which have the specified role.
func FindUsersAndSAsAffectedByPossibleRoleAddingOnGroupChange(txn *io.MemoryStoreTxn, oldGroup *model.Group, newGroup *model.Group, roleOfConcern string) (map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, error) {
	var err error

	// Group is deleted. There is no way to grant role on group deletion.
	if newGroup == nil {
		return nil, nil, nil
	}

	// New group is created. There is no role binding yet to grant role for users.
	if oldGroup == nil {
		return nil, nil, nil
	}

	// So, group is modified.
	// 1. Calculate newly added users. Return empty if there are no new users.
	// 2. Check role bindings for group and for parent groups.
	// 3. Return empty if there are no role bindings with role.
	// 4. Return newly added users and service accounts.

	// Calculate added users.
	addedUsers, addedSAs, addedGroups := calculateAddedSubjectsForGroup(oldGroup, newGroup)

	if len(addedUsers) == 0 && len(addedSAs) == 0 && len(addedGroups) == 0 {
		return nil, nil, nil
	}

	// Get all role binding for newGroup uuid and for all parent groups.
	groupRepo := model.NewGroupRepository(txn)
	parentGroups, err := groupRepo.FindAllParentGroupsForGroupUUID(newGroup.TenantUUID, newGroup.UUID)
	if err != nil {
		return nil, nil, err
	}

	hasRoleBindings, err := HasRoleBindingsWithGroupAndRole(txn, newGroup.TenantUUID, parentGroups, roleOfConcern)
	if err != nil {
		return nil, nil, fmt.Errorf("get role bindings with group and role '%s': %v", roleOfConcern, err)
	}

	// No affected members if there is no role bindings with the specified role.
	if !hasRoleBindings {
		return nil, nil, nil
	}

	// Return subjects for added users.
	return groupRepo.FindAllSubjectsFor(newGroup.TenantUUID, arrayFromMap(addedUsers), arrayFromMap(addedSAs), arrayFromMap(addedGroups))
}

// FindUsersAndSAsAffectedByPossibleRoleAddingOnGroupChange
// Return empty if group is added or deleted.
// Return empty if group is modified without adding new users.
// Return empty if group and parent groups have no role binding with the specified role.
// Return newly added users if group is modified and there are role bindings
// for group or for parent group which have the specified role.
func FindSubjectsAffectedByPossibleRoleAddingOnRoleChange(txn *io.MemoryStoreTxn, oldRole *model.Role, newRole *model.Role, roleOfConcern string) (map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, error) {
	// Role is deleted. There is no way to grant role on role deletion.
	if newRole == nil {
		return nil, nil, nil
	}

	// New role is created. There is no role binding yet to grant role for users.
	if oldRole == nil {
		return nil, nil, nil
	}

	oldRoleHasRole, err := CheckRoleObjectIncludeRole(txn, oldRole, roleOfConcern)
	if err != nil {
		return nil, nil, err
	}

	newRoleHasRole, err := CheckRoleObjectIncludeRole(txn, newRole, roleOfConcern)
	if err != nil {
		return nil, nil, err
	}

	if !oldRoleHasRole && newRoleHasRole {
		// Find subjects with changed role.
		return FindAllSubjectsWithGrantedRole(txn, newRole.Name)
	}

	// Role change does not affect users or service accounts.
	return nil, nil, nil
}

func arrayFromMap(m map[string]struct{}) []string {
	a := make([]string, 0)
	for k := range m {
		a = append(a, k)
	}
	return a
}

func CheckRoleBindingGrantsRole(txn *io.MemoryStoreTxn, roleBinding *model.RoleBinding, role string) (bool, error) {
	if roleBinding == nil {
		return false, nil
	}

	var err error
	var granted bool

	for _, includedRole := range roleBinding.Roles {
		granted, err = CheckRoleIncludeRole(txn, includedRole.Name, role)
		if err != nil {
			return false, err
		}
		if granted {
			break
		}
	}

	return granted, nil
}

func CheckRoleIncludeRole(txn *io.MemoryStoreTxn, role string, includedRole string) (bool, error) {
	if role == includedRole {
		return true, nil
	}

	repo := model.NewRoleRepository(txn)
	roles, err := repo.FindAllIncludingRoles(role)
	if err != nil {
		return false, err
	}

	for roleName := range roles {
		if roleName == includedRole {
			return true, nil
		}
	}

	return false, nil
}

func CheckRoleObjectIncludeRole(txn *io.MemoryStoreTxn, role *model.Role, includedRole string) (bool, error) {
	if includedRole == role.Name {
		return true, nil
	}

	repo := model.NewRoleRepository(txn)
	for _, included := range role.IncludedRoles {
		roles, err := repo.FindAllIncludingRoles(included.Name)
		if err != nil {
			return false, err
		}
		for roleName := range roles {
			if roleName == includedRole {
				return true, nil
			}
		}
	}

	return false, nil
}

func FindAllSubjectsWithGrantedRole(txn *io.MemoryStoreTxn, role model.RoleName) (map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, error) {
	parentRoles, err := GetParentRolesForRole(txn, role)
	if err != nil {
		return nil, nil, err
	}

	tenantRepo := model.NewTenantRepository(txn)
	tenants, err := tenantRepo.List()
	if err != nil {
		return nil, nil, err
	}

	repo := model.NewRoleBindingRepository(txn)
	roleBindings := make(map[model.RoleBindingUUID]struct{})
	for _, tenant := range tenants {
		tenantRoleBindings, err := repo.FindDirectRoleBindingsForRoles(tenant.UUID, parentRoles...)
		if err != nil {
			return nil, nil, err
		}
		for roleBindingUUID := range tenantRoleBindings {
			roleBindings[roleBindingUUID] = struct{}{}
		}
	}

	users := make(map[model.UserUUID]struct{})
	serviceAccounts := make(map[model.ServiceAccountUUID]struct{})

	groupRepo := model.NewGroupRepository(txn)
	for roleBinding := range roleBindings {
		rb, err := repo.GetByID(roleBinding)
		if err != nil {
			return nil, nil, err
		}
		rbUsers, rbSAs, err := groupRepo.FindAllSubjectsFor(rb.TenantUUID, rb.Users, rb.ServiceAccounts, rb.Groups)
		for uuid := range rbUsers {
			users[uuid] = struct{}{}
		}
		for uuid := range rbSAs {
			serviceAccounts[uuid] = struct{}{}
		}
	}

	return users, serviceAccounts, nil
}

func HasRoleBindingsWithGroupAndRole(txn *io.MemoryStoreTxn, tenantUUID model.TenantUUID, groups map[model.GroupUUID]struct{}, role string) (bool, error) {
	visited := make(map[model.RoleBindingUUID]struct{})

	repo := model.NewRoleBindingRepository(txn)
	for groupUUID := range groups {
		roleBindings, err := repo.FindDirectRoleBindingsForTenantGroups(tenantUUID, groupUUID)
		if err != nil {
			return false, err
		}

		for roleBindingUUID, roleBinding := range roleBindings {
			if _, has := visited[roleBindingUUID]; has {
				continue
			}
			hasRole, err := CheckRoleBindingHasRole(txn, roleBinding, role)
			if err != nil {
				return false, err
			}
			if hasRole {
				return true, nil
			}
			visited[roleBindingUUID] = struct{}{}
		}
	}

	return true, nil
}

func CheckRoleBindingHasRole(txn *io.MemoryStoreTxn, roleBinding *model.RoleBinding, role string) (bool, error) {
	roleBindingRoles := make(map[model.RoleName]struct{})
	for _, boundRole := range roleBinding.Roles {
		roleBindingRoles[boundRole.Name] = struct{}{}
	}

	// Check if role is directly in role binding.
	if _, has := roleBindingRoles[role]; has {
		return true, nil
	}

	// Check if role binding include one of parent roles.
	parentRoles, err := GetParentRolesForRole(txn, role)
	if err != nil {
		return false, err
	}

	for _, parentRole := range parentRoles {
		if _, has := roleBindingRoles[parentRole]; has {
			return true, nil
		}
	}

	return false, nil
}

func GetParentRolesForRole(txn *io.MemoryStoreTxn, role model.RoleName) ([]model.RoleName, error) {
	visited := make(map[model.RoleName]struct{})
	visited[role] = struct{}{}
	searchRoles := []model.RoleName{role}
	for {
		visitedNow := make(map[model.RoleName]struct{})
		for _, roleName := range searchRoles {
			iter, err := txn.Get(model.RoleType, "included_roles", roleName)
			if err != nil {
				return nil, err
			}
			for {
				raw := iter.Next()
				if raw == nil {
					break
				}
				role := raw.(*model.Role)
				if _, has := visited[role.Name]; !has {
					visitedNow[role.Name] = struct{}{}
				}
			}
		}

		// Stop if no roles visited.
		if len(visitedNow) == 0 {
			break
		}

		// Update visitedGroups and searchGroups for the next round.
		searchRoles = make([]model.RoleName, 0)
		for k := range visitedNow {
			visited[k] = struct{}{}
			searchRoles = append(searchRoles, k)
		}
	}

	return arrayFromMap(visited), nil
}

// Subjects

// calculateAddedSubjectsForGroup returns users and service accounts
// that was added in new version of the group.
func calculateAddedSubjectsForGroup(old *model.Group, new *model.Group) (map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, map[model.GroupUUID]struct{}) {
	return calculateAddedSubjects(
		old.Users, old.ServiceAccounts, old.Groups,
		new.Users, new.ServiceAccounts, new.Groups)
}

// calculateAddedSubjectsForRoleBinding returns users and service accounts
// that was added in new version of the group.
func calculateAddedSubjectsForRoleBinding(old *model.RoleBinding, new *model.RoleBinding) (map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, map[model.GroupUUID]struct{}) {
	return calculateAddedSubjects(
		old.Users, old.ServiceAccounts, old.Groups,
		new.Users, new.ServiceAccounts, new.Groups)
}

func calculateAddedSubjects(oldUsers []model.UserUUID, oldSAs []model.ServiceAccountUUID, oldGroups []model.GroupUUID, newUsers []model.UserUUID, newSAs []model.ServiceAccountUUID, newGroups []model.GroupUUID) (map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, map[model.GroupUUID]struct{}) {
	users := make(map[model.UserUUID]struct{})
	serviceAccounts := make(map[model.ServiceAccountUUID]struct{})
	groups := make(map[model.GroupUUID]struct{})
	for _, user := range oldUsers {
		users[user] = struct{}{}
	}
	for _, sa := range oldSAs {
		serviceAccounts[sa] = struct{}{}
	}
	for _, group := range oldGroups {
		groups[group] = struct{}{}
	}

	// Calculate added users.
	addedUsers := make(map[model.UserUUID]struct{})
	addedSAs := make(map[model.ServiceAccountUUID]struct{})
	addedGroups := make(map[model.GroupUUID]struct{})

	for _, user := range newUsers {
		if _, has := users[user]; !has {
			addedUsers[user] = struct{}{}
		}
	}
	for _, sa := range newSAs {
		if _, has := serviceAccounts[sa]; !has {
			addedSAs[sa] = struct{}{}
		}
	}
	for _, group := range newGroups {
		if _, has := groups[group]; !has {
			addedGroups[group] = struct{}{}
		}
	}

	return addedUsers, addedSAs, addedGroups
}
