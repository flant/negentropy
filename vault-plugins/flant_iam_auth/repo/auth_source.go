package repo

import (
	"errors"
	"fmt"
	"strings"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func AuthSourceSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.AuthSourceType: {
				Name: model.AuthSourceType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					ByName: {
						Name:   ByName,
						Unique: true,
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Name",
						},
					},
				},
			},
		},
	}
}

var (
	ErrSourceNotFound       = errors.New("ErrSourceNotFound")
	ErrSourceUsingInMethods = errors.New("ErrSourceUsingInMethods")
)

type AuthSourceRepo struct {
	db          io.Txn
	tableName   string
	methodsRepo *AuthMethodRepo
}

func NewAuthSourceRepo(db io.Txn) *AuthSourceRepo {
	return &AuthSourceRepo{
		db:          db,
		tableName:   model.AuthSourceType,
		methodsRepo: NewAuthMethodRepo(db),
	}
}

func (r *AuthSourceRepo) Get(name string) (*model.AuthSource, error) {
	raw, err := r.db.First(r.tableName, ByName, name)
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
	methods, err := r.methodsRepo.BySource(name)
	if err != nil {
		return err
	}

	if len(methods) > 0 {
		methodsNames := make([]string, 0)
		for _, m := range methods {
			methodsNames = append(methodsNames, m.Name)
		}

		return fmt.Errorf("%w can not delete source. use in [%s]", ErrSourceUsingInMethods, strings.Join(methodsNames, ","))
	}

	source, err := r.Get(name)
	if err != nil {
		return err
	}

	if source == nil {
		return ErrSourceNotFound
	}

	return r.db.Delete(r.tableName, source)
}

func (r *AuthSourceRepo) Iter(withInternal bool, action func(*model.AuthSource) (bool, error)) error {
	iter, err := r.db.Get(r.tableName, ID)
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
			return nil
		}
	}

	if withInternal {
		internals := []*model.AuthSource{
			model.GetMultipassSource(),
			model.GetServiceAccountPassSource(),
		}

		for _, s := range internals {
			next, err := action(s)
			if err != nil {
				return err
			}

			if !next {
				return nil
			}
		}
	}

	return nil
}
