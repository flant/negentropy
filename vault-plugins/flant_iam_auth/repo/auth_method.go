package repo

import (
	"fmt"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func AuthMethodSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.AuthMethodType: {
				Name: model.AuthMethodType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					ByName: {
						Name: ByName,
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Name",
						},
						Unique: true,
					},
				},
			},
		},
	}
}

type AuthMethodRepo struct {
	db        io.Txn
	tableName string
}

func NewAuthMethodRepo(db io.Txn) *AuthMethodRepo {
	return &AuthMethodRepo{
		db:        db,
		tableName: model.AuthMethodType,
	}
}

func (r *AuthMethodRepo) Get(name string) (*model.AuthMethod, error) {
	raw, err := r.db.First(r.tableName, ByName, name)
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

	return method, nil
}

func (r *AuthMethodRepo) BySource(name string) ([]*model.AuthMethod, error) {
	res := make([]*model.AuthMethod, 0)
	// because source name may be empty we don't use index
	err := r.Iter(func(method *model.AuthMethod) (bool, error) {
		if method.Source == name {
			res = append(res, method)
		}
		return true, nil
	})
	return res, err
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
	iter, err := r.db.Get(r.tableName, ID)
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
