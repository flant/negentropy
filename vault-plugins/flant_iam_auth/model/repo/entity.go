package repo

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type EntityRepo struct {
	db        *io.MemoryStoreTxn
	tableName string
}

func NewEntityRepo(db *io.MemoryStoreTxn) *EntityRepo {
	return &EntityRepo{
		db:        db,
		tableName: model.EntityType,
	}
}

func (r *EntityRepo) GetByID(id string) (*model.Entity, error) {
	return r.get(model.ID, id)
}

func (r *EntityRepo) GetByUserId(name string) (*model.Entity, error) {
	return r.get(model.ByUserID, name)
}

func (r *EntityRepo) get(by string, val string) (*model.Entity, error) {
	raw, err := r.db.First(r.tableName, by, val)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	source, ok := raw.(*model.Entity)
	if !ok {
		return nil, fmt.Errorf("cannot cast to Entity")
	}

	return source, nil
}

func (r *EntityRepo) Put(source *model.Entity) error {
	return r.db.Insert(r.tableName, source)
}

func (r *EntityRepo) DeleteByID(id string) error {
	source, err := r.get(model.ID, id)
	if err != nil {
		return err
	}
	return r.db.Delete(r.tableName, source)
}

func (r *EntityRepo) List() ([]string, error) {
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
		t := raw.(*model.Entity)
		ids = append(ids, t.Name)
	}
	return ids, nil
}
