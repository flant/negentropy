package internal

import (
	"errors"
	"fmt"

	"github.com/hashicorp/go-hclog"

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
		ObjType:    iam_model.TenantType,
		CallbackFn: h.processTenant,
	})
	memstorage.RegisterHook(sharedio.ObjectHook{
		Events:     []sharedio.HookEvent{sharedio.HookEventInsert}, // only process insert, as use archiving for this item
		ObjType:    iam_model.ProjectType,
		CallbackFn: h.processProject,
	})
	memstorage.RegisterHook(sharedio.ObjectHook{
		Events:     []sharedio.HookEvent{sharedio.HookEventInsert}, // only process insert, as use archiving for this item
		ObjType:    iam_model.UserType,
		CallbackFn: h.processUser,
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

func (h *Hooker) processTenant(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, obj interface{}) error {
	h.Logger.Debug("call processTenant")
	h.Logger.Debug(fmt.Sprintf("%#v\n", obj))
	// TODO
	return nil
}

func (h *Hooker) processProject(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, obj interface{}) error {
	h.Logger.Debug("call processProject")
	h.Logger.Debug(fmt.Sprintf("%#v\n", obj))
	// TODO
	return nil
}

func (h *Hooker) processUser(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, obj interface{}) error {
	h.Logger.Debug("call processUser")
	h.Logger.Debug(fmt.Sprintf("%#v\n", obj))
	// TODO
	return nil
}

func (h *Hooker) processGroup(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, obj interface{}) error {
	h.Logger.Debug("call processGroup")
	h.Logger.Debug(fmt.Sprintf("%#v\n", obj))
	// TODO
	return nil
}

func (h *Hooker) processRolebinding(txn *sharedio.MemoryStoreTxn, _ sharedio.HookEvent, objNewRolebinding interface{}) error {
	h.Logger.Debug("call processRolebinding")
	h.Logger.Debug(fmt.Sprintf("new rolebinding %#v\n", objNewRolebinding))
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
	err = h.processor.UpdateUserEffectiveRoles(txn, allPossibleChangedUsers, allPossibleChangedRoles)
	if err != nil {
		return err
	}
	return nil
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

func collectAllUsers(txn *sharedio.MemoryStoreTxn, rolebindings ...*iam_model.RoleBinding) (map[iam_model.UserUUID]struct{}, error) {
	users := map[iam_model.UserUUID]struct{}{}
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
