package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
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
