package repo

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type AuthMethodRepo struct {
	db        *io.MemoryStoreTxn
	tableName string
}

func NewAuthMethodRepo(db *io.MemoryStoreTxn) *AuthMethodRepo {
	return &AuthMethodRepo{
		db:        db,
		tableName: model.AuthMethodType,
	}
}

func (r *AuthMethodRepo) Get(name string) (*model.AuthMethod, error) {
	raw, err := r.db.First(r.tableName, model.ByName, name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	method, ok := raw.(*model.AuthMethod)
	if !ok {
		return nil, fmt.Errorf("cannot cast to AuthMethod")
	}

	if method.BoundClaimsType == "" {
		method.BoundClaimsType = model.BoundClaimsTypeString
	}

	return method, nil
}

func (r *AuthMethodRepo) Put(source *model.AuthMethod) error {
	return r.db.Insert(r.tableName, source)
}

func (r *AuthMethodRepo) Delete(name string) error {
	method, err := r.Get(name)
	if err != nil {
		return err
	}
	return r.db.Delete(r.tableName, method)
}

func (r *AuthMethodRepo) Iter(action func(*model.AuthMethod) (bool, error)) error {
	iter, err := r.db.Get(r.tableName, model.ID)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.AuthMethod)
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
