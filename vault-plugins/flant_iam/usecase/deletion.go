package usecase

import (
	"math/rand"
	"time"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type DeleterByParent interface {
	DeleteByParent(string, model.UnixTime, int64) error
}

func deleteChildren(parentID string, deleters []DeleterByParent, archivingTimestamp model.UnixTime, archivingHash int64) error {
	for _, d := range deleters {
		err := d.DeleteByParent(parentID, archivingTimestamp, archivingHash)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListerDeleter is the interface for repos to comply with the deletion case
type ListerDeleter interface {
	ListIDs(parentID string, showArchived bool) ([]string, error)
	Delete(id string, archivingTimestamp model.UnixTime, archivingHash int64) error
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
func (d *ChildrenDeleter) DeleteByParent(parentID string, archivingTimestamp model.UnixTime, archivingHash int64) error {
	ids, err := d.childrenDeleter.ListIDs(parentID, false)
	if err != nil {
		return err
	}
	for _, childID := range ids {
		if err := deleteChildren(childID, d.grandChildrenDeleters, archivingTimestamp, archivingHash); err != nil {
			return err
		}
		if err := d.childrenDeleter.Delete(childID, archivingTimestamp, archivingHash); err != nil {
			return err
		}
	}
	return nil
}

// func MultipassDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
//	return NewChildrenDeleter(
//		repo.NewMultipassRepository(tx),
//	)
// }
//
// func PasswordDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
//	return NewChildrenDeleter(
//		repo.NewServiceAccountPasswordRepository(tx),
//	)
// }

func RoleBindingApprovalDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
	return NewChildrenDeleter(
		repo.NewRoleBindingApprovalRepository(tx),
	)
}

// func RoleBindingDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
//	return NewChildrenDeleter(
//		repo.NewRoleBindingRepository(tx),
//		RoleBindingApprovalDeleter(tx),
//	)
// }

// func NewIdentitySharingDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
//	return NewChildrenDeleter(
//		repo.NewIdentitySharingRepository(tx),
//		// TODO clean identity sharings where the tenant is the destination
//	)
// }

// func ServiceAccountDeleter(tx *io.MemoryStoreTxn) *ChildrenDeleter {
//	return NewChildrenDeleter(
//		repo.NewServiceAccountRepository(tx),
//		MultipassDeleter(tx),
//		PasswordDeleter(tx),
//		// TODO clean SA references from rolebindings and groups in other tenants
//	)
// }

func ArchivingLabel() (model.UnixTime, int64) {
	archivingTime := time.Now().Unix()
	archivingHash := rand.Int63n(archivingTime)
	return archivingTime, archivingHash
}
