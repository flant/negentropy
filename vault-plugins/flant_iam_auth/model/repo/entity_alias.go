package repo

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type EntityAliasRepo struct {
	db        *io.MemoryStoreTxn
	tableName string
}

func NewEntityAliasRepo(db *io.MemoryStoreTxn) *EntityAliasRepo {
	return &EntityAliasRepo{
		db:        db,
		tableName: model.EntityAliasType,
	}
}

func (r *EntityAliasRepo) GetByID(id string) (*model.EntityAlias, error) {
	return r.get(model.ID, id)
}

func (r *EntityAliasRepo) GetByUserID(id string, sourceName string) (*model.EntityAlias, error) {
	return r.get(model.ByUserID, id)
}

func (r *EntityAliasRepo) get(by string, val string) (*model.EntityAlias, error) {
	raw, err := r.db.First(r.tableName, by, val)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	source, ok := raw.(*model.EntityAlias)
	if !ok {
		return nil, fmt.Errorf("cannot cast to EntityAlias")
	}

	return source, nil
}

func (r *EntityAliasRepo) Put(source *model.EntityAlias) error {
	return r.db.Insert(r.tableName, source)
}

func (r *EntityAliasRepo) DeleteByID(id string) error {
	source, err := r.get(model.ID, id)
	if err != nil {
		return err
	}
	return r.db.Delete(r.tableName, source)
}

func (r *EntityAliasRepo) List() ([]string, error) {
	iter, err := r.db.Get(r.tableName, model.ID)
	if err != nil {
		return nil, err
	}

	var ids []string
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.EntityAlias)
		ids = append(ids, t.Name)
	}
	return ids, nil
}
