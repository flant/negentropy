package usecase

import "github.com/flant/negentropy/vault-plugins/flant_iam/model"

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
	ListIDs(string) ([]string, error)
	Delete(string) error
}

func NewChildrenDeleter(repo ListerDeleter, deleters ...DeleterByParent) *ChildrenDeleter {
	return &ChildrenDeleter{
		repo:     repo,
		deleters: deleters,
	}
}

// ChildrenDeleter unifies deleting a model object with its children
type ChildrenDeleter struct {
	repo     ListerDeleter
	deleters []DeleterByParent
}

// DeleteByParent deletes children objects and then the parent one
func (d *ChildrenDeleter) DeleteByParent(parentID string) error {
	ids, err := d.repo.ListIDs(parentID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := deleteChildren(id, d.deleters); err != nil {
			return err
		}

	}
	return nil
}
