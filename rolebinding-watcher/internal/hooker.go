// How it works
// The main source of UserEffectiveRoles changes is Rolebinding,
// Rolebinding: 1) New/Update/Archive - need be processed 2) Delete - doesn't matter as after archiving UserEffectiveRoles disappear
// User: 1) New - new user hasn't any Rolebinding 2) Update - doesn't change anything 3) Archive/Delete - this user can't have any Active Rolebinding
// Tenant: 1) New - new tenant hasn't any Rolebinding 2) Update - doesn't change anything 3) Archive/Delete - this tenant can't have any Active Rolebinding
// Group: 1) New - new group hasn't any Rolebinding 2) Archive/Delete - this group can't have any Active Rolebinding 3) Update - if was changed set of users/or group it can change usereffectiveRoles, but if kafka will be compacted, new item can be not new, but edited
//        need to be processed all roles which are on old and new group.
// Project: 1) New/Archive - project can change userEffectiveRole under projectScopedRoles 2) Update/Delete doesn't produce any changes

package internal

import (
	"errors"
	"fmt"

	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/rolebinding-watcher/pkg"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	sharedio "github.com/flant/negentropy/vault-plugins/shared/io"
)

type Hooker struct {
	Logger    hclog.Logger
	processor *ChangesProcessor
}

func (h *Hooker) RegisterHooks(memstorage *sharedio.MemoryStore) {
	memstorage.RegisterHook(sharedio.ObjectHook{
		Events:     []sharedio.HookEvent{sharedio.HookEventInsert}, // only process insert, as use archiving for this item
		ObjType:    iam_model.ProjectType,
		CallbackFn: h.processProject,
	})
	memstorage.RegisterHook(sharedio.ObjectHook{
		Events:     []sharedio.HookEvent{sharedio.HookEventInsert}, // only process insert, as use archiving for this item
		ObjType:    iam_model.GroupType,
		CallbackFn: h.processGroup,
	})
	memstorage.RegisterHook(sharedio.ObjectHook{
		Events:     []sharedio.HookEvent{sharedio.HookEventInsert}, // only process insert, as use archiving for this item
		ObjType:    iam_model.RoleBindingType,
		CallbackFn: h.processRolebinding,
	})
	memstorage.RegisterHook(sharedio.ObjectHook{
		Events:     []sharedio.HookEvent{sharedio.HookEventInsert}, // only process insert, as use archiving for this item
		ObjType:    iam_model.RoleType,
		CallbackFn: h.processRole,
	})
}

func (h *Hooker) processProject(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, objNewProject interface{}) error {
	h.Logger.Debug("call processProject")
	newProject, ok := objNewProject.(*iam_model.Project)
	if !ok {
		return fmt.Errorf("%w: expected type *iam_model.Project, got: %T", consts.CriticalCodeError, objNewProject)
	}
	oldProject, err := iam_repo.NewProjectRepository(txn).GetByID(newProject.UUID)
	if errors.Is(err, consts.ErrNotFound) {
		err = nil
	}
	if !((oldProject == nil && newProject.NotArchived()) || // new active project adding
		(oldProject != nil && oldProject.NotArchived() && newProject.Archived())) { // archiving project
		return nil // nothing happen
	}
	rbs, err := iam_repo.NewRoleBindingRepository(txn).List(newProject.TenantUUID, false)
	if err != nil {
		return fmt.Errorf("collecting tenant rolebindings: %w", err)
	}
	allPossibleChangedUsers, err := collectUsers(txn, rbs)
	if err != nil {
		return err
	}
	allPossibleChangedRoles, err := collectAllProjectScopedRolesFromRolebindings(txn, rbs)
	if err != nil {
		return err
	}

	err = txn.Txn.Insert(newProject.ObjType(), newProject) // It's a dirty hack to get future state of DB TODO remake it with writing own store with hooks recieving old, new objects and future txn
	if err != nil {
		return err
	}
	return h.processor.UpdateUserEffectiveRoles(txn, allPossibleChangedUsers, allPossibleChangedRoles)
}

func collectUsers(txn *sharedio.MemoryStoreTxn, rbs []*iam_model.RoleBinding) (map[pkg.UserUUID]struct{}, error) {
	// collect all direct groups and users
	groupSet := map[iam_model.GroupUUID]struct{}{}
	userSet := map[pkg.UserUUID]struct{}{}
	for _, rb := range rbs {
		for _, g := range rb.Groups {
			groupSet[g] = struct{}{}
		}
		for _, u := range rb.Users {
			userSet[u] = struct{}{}
		}
	}
	users, _, err := iam_repo.NewGroupRepository(txn).FindAllMembersFor(makeSlice(userSet), nil, makeSlice(groupSet))
	return users, err
}

func (h *Hooker) processGroup(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, objNewGroup interface{}) error {
	h.Logger.Debug("call processGroup")
	newGroup, ok := objNewGroup.(*iam_model.Group)
	if !ok {
		return fmt.Errorf("%w: expected type *iam_model.Group, got: %T", consts.CriticalCodeError, objNewGroup)
	}
	oldGroup, err := iam_repo.NewGroupRepository(txn).GetByID(newGroup.UUID)
	if errors.Is(err, consts.ErrNotFound) {
		err = nil
	}
	if newGroup.Archived() { // nothing happen
		return nil
	}
	if oldGroup == nil {
		oldGroup = &iam_model.Group{}
	}
	// only changing group is processed - collect diff users of both group
	allPossibleChangedUsers, err := collectUserDiff(txn, *newGroup, *oldGroup)
	if err != nil {
		return err
	}

	allPossibleChangedRoles, err := collectAllRolesOfGroups(txn, append(newGroup.Groups, oldGroup.Groups...))
	if err != nil {
		return err
	}

	err = txn.Txn.Insert(newGroup.ObjType(), newGroup) // It's a dirty hack to get future state of DB TODO remake it with writing own store with hooks recieving old, new objects and future txn
	if err != nil {
		return err
	}
	return h.processor.UpdateUserEffectiveRoles(txn, allPossibleChangedUsers, allPossibleChangedRoles)
}

func collectAllRolesOfGroups(txn *sharedio.MemoryStoreTxn, groups []iam_model.GroupUUID) (map[pkg.RoleName]struct{}, error) {
	// build allGroupSet which rolebindings can produce changes
	groupsSet := map[iam_model.GroupUUID]struct{}{}
	for _, g := range groups {
		groupsSet[g] = struct{}{}
	}
	allGroups, err := iam_repo.NewGroupRepository(txn).FindAllParentGroupsForGroupUUIDs(groupsSet)
	if err != nil {
		return nil, fmt.Errorf("collecting groups: %w", err)
	}
	// collect roles
	rbs, err := iam_repo.NewRoleBindingRepository(txn).FindDirectRoleBindingsForGroups(makeSlice(allGroups)...)
	roles, err := collectAllRolesFromRolebindings(txn, rbs)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func collectAllRolesFromRolebindings(txn *sharedio.MemoryStoreTxn, rbs map[iam_model.RoleBindingUUID]*iam_model.RoleBinding) (map[iam_model.RoleName]struct{}, error) {
	roles := map[iam_model.RoleName]struct{}{}
	roleRepo := iam_repo.NewRoleRepository(txn)
	for _, rb := range rbs {
		// range over roles
		for _, role := range rb.Roles {
			roles[role.Name] = struct{}{}
			childRoles, err := roleRepo.FindAllChildrenRoles(role.Name)
			if err != nil {
				return nil, fmt.Errorf("collecting child roles: %w", err)
			}
			for roleName := range childRoles {
				roles[roleName] = struct{}{}
			}
		}
	}
	return roles, nil
}

func collectAllProjectScopedRolesFromRolebindings(txn *sharedio.MemoryStoreTxn, rbs []*iam_model.RoleBinding) (map[iam_model.RoleName]struct{}, error) {
	roles := map[iam_model.RoleName]struct{}{}
	roleRepo := iam_repo.NewRoleRepository(txn)
	for _, rb := range rbs {
		// range over roles
		for _, role := range rb.Roles {
			roles[role.Name] = struct{}{}
			childRoles, err := roleRepo.FindAllChildrenRoles(role.Name)
			if err != nil {
				return nil, fmt.Errorf("collecting child roles: %w", err)
			}
			for roleName, role := range childRoles {
				if role.Scope == iam_model.RoleScopeProject {
					roles[roleName] = struct{}{}
				}
			}
		}
	}
	return roles, nil
}

func collectUserDiff(txn *sharedio.MemoryStoreTxn, newGroup iam_model.Group, oldGroup iam_model.Group) (map[pkg.UserUUID]struct{}, error) {
	groupDiff := map[iam_model.GroupUUID]struct{}{}
	for _, ng := range newGroup.Groups {
		groupDiff[ng] = struct{}{}
	}
	for _, og := range oldGroup.Groups {
		if _, has := groupDiff[og]; has {
			delete(groupDiff, og)
		} else {
			groupDiff[og] = struct{}{}
		}
	}

	userDiff := map[pkg.UserUUID]struct{}{}
	for _, nu := range newGroup.Users {
		userDiff[nu] = struct{}{}
	}
	for _, ou := range oldGroup.Users {
		if _, has := userDiff[ou]; has {
			delete(userDiff, ou)
		} else {
			userDiff[ou] = struct{}{}
		}
	}
	users, _, err := iam_repo.NewGroupRepository(txn).FindAllMembersFor(makeSlice(userDiff), nil, makeSlice(groupDiff))
	return users, err
}

func (h *Hooker) processRolebinding(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, objNewRolebinding interface{}) error {
	newRolebinding, ok := objNewRolebinding.(*iam_model.RoleBinding)
	if !ok {
		return fmt.Errorf("%w: expected type *iam_model.RoleBinding, got: %T", consts.CriticalCodeError, objNewRolebinding)
	}
	oldRolebinding, err := iam_repo.NewRoleBindingRepository(txn).GetByID(newRolebinding.UUID)
	if errors.Is(err, consts.ErrNotFound) {
		err = nil
	}
	if oldRolebinding != nil && newRolebinding.Archived() && oldRolebinding.Archived() { // nothing happen
		return nil
	}
	allPossibleChangedRoles, err := collectChildrenRoles(txn, newRolebinding, oldRolebinding)
	if err != nil {
		return err
	}
	allPossibleChangedUsers, err := collectAllUsers(txn, newRolebinding, oldRolebinding)
	if err != nil {
		return err
	}
	err = txn.Txn.Insert(newRolebinding.ObjType(), newRolebinding) // It's a dirty hack to get future state of DB TODO remake it with writing own store with hooks recieving old, new objects and future txn
	if err != nil {
		return err
	}
	return h.processor.UpdateUserEffectiveRoles(txn, allPossibleChangedUsers, allPossibleChangedRoles)
}

func (h *Hooker) processRole(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, obj interface{}) error {
	h.Logger.Debug("call processRole")
	h.Logger.Debug(fmt.Sprintf("%#v\n", obj))
	// TODO
	return nil
}

// add to roles all roles from rolebinding
func collectChildrenRoles(txn *sharedio.MemoryStoreTxn, rolebindings ...*iam_model.RoleBinding) (map[iam_model.RoleName]struct{}, error) {
	roles := map[iam_model.RoleName]struct{}{}
	repo := iam_repo.NewRoleRepository(txn)
	for _, rolebinding := range rolebindings {
		if rolebinding == nil {
			continue
		}
		for _, r := range rolebinding.Roles {
			extraRoles, err := repo.FindAllChildrenRoles(r.Name)
			if err != nil {
				return nil, err
			}
			for roleName := range extraRoles {
				roles[roleName] = struct{}{}
			}
		}
	}
	return roles, nil
}

func collectAllUsers(txn *sharedio.MemoryStoreTxn, rolebindings ...*iam_model.RoleBinding) (map[pkg.UserUUID]struct{}, error) {
	users := map[pkg.UserUUID]struct{}{}
	repo := iam_repo.NewGroupRepository(txn)
	for _, rolebinding := range rolebindings {
		if rolebinding == nil {
			continue
		}
		newUsers, _, err := repo.FindAllMembersFor(rolebinding.Users, nil, rolebinding.Groups)
		if err != nil {
			return nil, err
		}
		for userUUID := range newUsers {
			users[userUUID] = struct{}{}
		}
	}
	return users, nil
}
