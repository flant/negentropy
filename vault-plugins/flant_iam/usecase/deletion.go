package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type DeleterByParent interface {
	DeleteByParent(string) error
}

func deleteChildren(parentID string, deleters []DeleterByParent) error {
	for _, d := range deleters {
		err := d.DeleteByParent(parentID)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListerDeleter is the interface for repos to comply with the deletion case
type ListerDeleter interface {
	ListIDs(parentID string) ([]string, error)
	Delete(id string) error
}

func NewChildrenDeleter(repo ListerDeleter, deleters ...DeleterByParent) *ChildrenDeleter {
	return &ChildrenDeleter{
		childrenDeleter:       repo,
		grandChildrenDeleters: deleters,
	}
}

// ChildrenDeleter unifies deleting a model object with its children
type ChildrenDeleter struct {
	childrenDeleter       ListerDeleter
	grandChildrenDeleters []DeleterByParent
}

// DeleteByParent deletes children objects and then the parent one
func (d *ChildrenDeleter) DeleteByParent(parentID string) error {
	ids, err := d.childrenDeleter.ListIDs(parentID)
	if err != nil {
		return err
	}
	for _, childID := range ids {
		if err := deleteChildren(childID, d.grandChildrenDeleters); err != nil {
			return err
		}
		if err := d.childrenDeleter.Delete(childID); err != nil {
			return err
		}
	}
	return nil
}

func MultipassDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewMultipassRepository(tx),
	)
}

func PasswordDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewServiceAccountPasswordRepository(tx),
	)
}

func GroupDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewGroupRepository(tx),
	)
}

func RoleBindingApprovalDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewRoleBindingRepository(tx),
	)
}

func RoleBindingDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewRoleBindingRepository(tx),
		RoleBindingApprovalDeleter(tx),
	)
}

func ProjectDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewProjectRepository(tx),
	)
}

func NewIdentitySharingDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewIdentitySharingRepository(tx),
	)
}

func UserDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewUserRepository(tx),
		MultipassDeleter(tx),
	)
}

func ServiceAccountDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		model.NewServiceAccountRepository(tx),
		MultipassDeleter(tx),
		PasswordDeleter(tx),
	)
}
