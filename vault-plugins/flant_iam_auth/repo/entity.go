package repo

import (
	"fmt"

	hcmemdb "github.com/hashicorp/go-memdb"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

func EntitySchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.EntityType: {
				Name: model.EntityType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					ByUserID: {
						Name:   ByUserID,
						Unique: true,
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "UserId",
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

type EntityRepo struct {
	db io.Txn
}

func NewEntityRepo(db io.Txn) *EntityRepo {
	return &EntityRepo{
		db: db,
	}
}

func (r *EntityRepo) GetByID(id string) (*model.Entity, error) {
	return r.get(ID, id)
}

func (r *EntityRepo) GetByName(name string) (*model.Entity, error) {
	return r.get(ByName, name)
}

func (r *EntityRepo) GetByUserId(userID string) (*model.Entity, error) {
	return r.get(ByUserID, userID)
}

func (r *EntityRepo) CreateForUser(user *iam.User) error {
	return r.putNew(user.FullIdentifier, user.UUID)
}

func (r *EntityRepo) CreateForSA(sa *iam.ServiceAccount) error {
	return r.putNew(sa.FullIdentifier, sa.UUID)
}

func (r *EntityRepo) get(by string, val string) (*model.Entity, error) {
	raw, err := r.db.First(model.EntityType, by, val)
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

func (r *EntityRepo) putNew(name string, userId string) error {
	entity, err := r.GetByUserId(userId)
	if err != nil {
		return err
	}

	if entity != nil {
		return nil
	}

	entity = &model.Entity{
		UUID:   utils.UUID(),
		UserId: userId,
	}
	entity.Name = name

	return r.db.Insert(model.EntityType, entity)
}

func (r *EntityRepo) Put(source *model.Entity) error {
	return r.db.Insert(model.EntityType, source)
}

func (r *EntityRepo) DeleteForUser(id string) error {
	source, err := r.get(ByUserID, id)
	if err != nil {
		return err
	}
	return r.db.Delete(model.EntityType, source)
}
