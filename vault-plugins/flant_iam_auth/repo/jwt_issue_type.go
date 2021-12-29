package repo

import (
	"fmt"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func JWTIssueTypeSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.JWTIssueTypeType: {
				Name: model.JWTIssueTypeType,
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
					},
				},
			},
		},
	}
}

type JWTIssueTypeRepo struct {
	db        *io.MemoryStoreTxn
	tableName string
}

func NewJWTIssueTypeRepo(db *io.MemoryStoreTxn) *JWTIssueTypeRepo {
	return &JWTIssueTypeRepo{
		db:        db,
		tableName: model.JWTIssueTypeType,
	}
}

func (r *JWTIssueTypeRepo) Get(name string) (*model.JWTIssueType, error) {
	raw, err := r.db.First(r.tableName, ByName, name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	method, ok := raw.(*model.JWTIssueType)
	if !ok {
		return nil, fmt.Errorf("cannot cast to JWTIssueType")
	}

	return method, nil
}

func (r *JWTIssueTypeRepo) Put(source *model.JWTIssueType) error {
	return r.db.Insert(r.tableName, source)
}

func (r *JWTIssueTypeRepo) Delete(name string) error {
	method, err := r.Get(name)
	if err != nil {
		return err
	}
	return r.db.Delete(r.tableName, method)
}

func (r *JWTIssueTypeRepo) Iter(action func(*model.JWTIssueType) (bool, error)) error {
	iter, err := r.db.Get(r.tableName, ID)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.JWTIssueType)
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
