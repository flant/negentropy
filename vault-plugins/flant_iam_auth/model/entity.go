package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

const (
	EntityType = "entity" // also, memdb schema name
)

type Entity struct {
	UUID   string `json:"uuid"` // ID
	Name   string `json:"name"` // Identifier
	UserId string `json:"user_id"`
}

func (p *Entity) ObjType() string {
	return EntityType
}

func (p *Entity) ObjId() string {
	return p.UUID
}

func EntitySchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			EntityType: {
				Name: EntityType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					ByUserID: {
						Name:   ByUserID,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "UserId",
						},
					},
				},
			},
		},
	}
}

type EntityRepo struct {
	db        *io.MemoryStoreTxn
	tableName string
}

func NewEntityRepo(db *io.MemoryStoreTxn) *EntityRepo {
	return &EntityRepo{
		db:        db,
		tableName: EntityType,
	}
}

func (r *EntityRepo) GetByID(id string) (*Entity, error) {
	return r.get(ID, id)
}

func (r *EntityRepo) GetByUserId(name string) (*Entity, error) {
	return r.get(ByUserID, name)
}

func (r *EntityRepo) CreateForUser(user *iam.User) error {
	return r.putNew(user.FullIdentifier, user.UUID)
}

func (r *EntityRepo) CreateForSA(sa *iam.ServiceAccount) error {
	return r.putNew(sa.FullIdentifier, sa.UUID)
}

func (r *EntityRepo) get(by string, val string) (*Entity, error) {
	raw, err := r.db.First(r.tableName, by, val)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	source, ok := raw.(*Entity)
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

	entity = &Entity{
		UUID:   utils.UUID(),
		UserId: userId,
	}
	entity.Name = name

	return r.db.Insert(r.tableName, entity)
}

func (r *EntityRepo) Put(source *Entity) error {
	return r.db.Insert(r.tableName, source)
}

func (r *EntityRepo) DeleteForUser(id string) error {
	source, err := r.get(ByUserID, id)
	if err != nil {
		return err
	}
	return r.db.Delete(r.tableName, source)
}
