package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

const (
	EntityAliasType   = "entity_alias" // also, memdb schema name
	EntityAliasSource = "entity_alias_source"
	EntityName        = "entity_name"

	BySourceId = "source_id"
)

type EntityAlias struct {
	UUID     string `json:"uuid"`    // ID
	UserId   string `json:"user_id"` // user is user or sa or multipass
	Name     string `json:"name"`    // source name. by it vault look alias for user
	SourceId string `json:"source_id"`
}

func (p *EntityAlias) ObjType() string {
	return EntityAliasType
}

func (p *EntityAlias) ObjId() string {
	return p.UUID
}

func EntityAliasSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			EntityAliasType: {
				Name: EntityAliasType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					EntityAliasSource: {
						Name: EntityAliasSource,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{
									Field: "UserId",
								},

								&memdb.StringFieldIndex{
									Field: "SourceId",
								},
							},
						},
					},
					ByName: {
						Name: ByName,
						Indexer: &memdb.StringFieldIndex{
							Field: "SourceId",
						},
					},

					BySourceId: {
						Name: BySourceId,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},

					ByUserID: {
						Name: ByUserID,
						Indexer: &memdb.StringFieldIndex{
							Field: "UserId",
						},
					},
				},
			},
		},
	}
}

type EntityAliasRepo struct {
	db        *io.MemoryStoreTxn
	tableName string
}

func NewEntityAliasRepo(db *io.MemoryStoreTxn) *EntityAliasRepo {
	return &EntityAliasRepo{
		db:        db,
		tableName: EntityAliasType,
	}
}

func (r *EntityAliasRepo) GetById(id string) (*EntityAlias, error) {
	return r.get(ID, id)
}

func (r *EntityAliasRepo) GetAllForUser(id string, action func(*EntityAlias) (bool, error)) error {
	return r.iter(action, ByUserID, id)
}

func (r *EntityAliasRepo) GetBySource(sourceUUID string, action func(*EntityAlias) (bool, error)) error {
	return r.iter(action, BySourceId, sourceUUID)
}

func (r *EntityAliasRepo) GetForUser(id string, source *AuthSource) (*EntityAlias, error) {
	return r.get(EntityAliasSource, id, source.UUID)
}

func (r *EntityAliasRepo) get(by string, vals ...interface{}) (*EntityAlias, error) {
	raw, err := r.db.First(r.tableName, by, vals...)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	source, ok := raw.(*EntityAlias)
	if !ok {
		return nil, fmt.Errorf("cannot cast to EntityAlias")
	}

	return source, nil
}

func (r *EntityAliasRepo) Put(source *EntityAlias) error {
	return r.db.Insert(r.tableName, source)
}

func (r *EntityAliasRepo) CreateForUser(user *iam.User, source *AuthSource) error {
	name, err := source.NameForUser(user)
	if err != nil {
		return err
	}

	if name == "" {
		return fmt.Errorf("empty ea name for source %s and user %s/%s/%s", source.Name, user.UUID, user.FullIdentifier, user.Email)
	}

	err = r.putNew(user.UUID, source, name)
	if err != nil {
		return err
	}

	return nil
}

func (r *EntityAliasRepo) CreateForSA(sa *iam.ServiceAccount, source *AuthSource) error {
	// skip no sa
	if !source.AllowForSA() {
		return nil
	}

	name, err := source.NameForServiceAccount(sa)
	if err != nil {
		return err
	}

	return r.putNew(sa.UUID, source, name)
}

func (r *EntityAliasRepo) DeleteByID(id string) error {
	source, err := r.get(ID, id)
	if err != nil {
		return err
	}
	return r.db.Delete(r.tableName, source)
}

func (r *EntityAliasRepo) List() ([]string, error) {
	iter, err := r.db.Get(r.tableName, ID)
	if err != nil {
		return nil, err
	}

	var ids []string
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*EntityAlias)
		ids = append(ids, t.Name)
	}
	return ids, nil
}

func (r *EntityAliasRepo) iter(action func(alias *EntityAlias) (bool, error), key string, args interface{}) error {
	iter, err := r.db.Get(r.tableName, key, args)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*EntityAlias)
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

func (r *EntityAliasRepo) putNew(iamEntityId string, source *AuthSource, eaName string) error {
	sourceId := source.UUID

	entityAlias, err := r.GetForUser(iamEntityId, source)
	if err != nil {
		return err
	}

	if entityAlias == nil {
		entityAlias = &EntityAlias{
			UUID:     utils.UUID(),
			UserId:   iamEntityId,
			SourceId: sourceId,
		}
	}

	entityAlias.Name = eaName

	return r.db.Insert(r.tableName, entityAlias)
}
