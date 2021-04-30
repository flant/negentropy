package repo

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type AuthSourceRepo struct {
	db        *io.MemoryStoreTxn
	tableName string
}

func NewAuthSourceRepo(db *io.MemoryStoreTxn) *AuthSourceRepo {
	return &AuthSourceRepo{
		db:        db,
		tableName: model.AuthSourceType,
	}
}

func (r *AuthSourceRepo) Get(name string) (*model.AuthSource, error) {
	raw, err := r.db.First(r.tableName, model.ByName, name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	source, ok := raw.(*model.AuthSource)
	if !ok {
		return nil, fmt.Errorf("cannot cast to AuthSource")
	}

	err = source.PopulatePubKeys()
	if err != nil {
		return nil, err
	}

	return source, nil
}

func (r *AuthSourceRepo) Put(source *model.AuthSource) error {
	return r.db.Insert(r.tableName, source)
}

func (r *AuthSourceRepo) Delete(name string) error {
	source, err := r.Get(name)
	if err != nil {
		return err
	}
	return r.db.Delete(r.tableName, source)
}

func (r *AuthSourceRepo) Iter(action func(*model.AuthSource) (bool, error)) error {
	iter, err := r.db.Get(r.tableName, model.ID)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.AuthSource)
		next, err := action(t)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
}
