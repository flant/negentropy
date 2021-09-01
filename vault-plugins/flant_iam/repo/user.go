package repo

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func UserSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			model.UserType: {
				Name: model.UserType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					"version": {
						Name: "version",
						Indexer: &memdb.StringFieldIndex{
							Field: "Version",
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &memdb.StringFieldIndex{
							Field: "Identifier",
						},
					},
				},
			},
		},
	}
}

type UserRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewUserRepository(tx *io.MemoryStoreTxn) *UserRepository {
	return &UserRepository{db: tx}
}

func (r *UserRepository) save(user *model.User) error {
	return r.db.Insert(model.UserType, user)
}

func (r *UserRepository) Create(user *model.User) error {
	return r.save(user)
}

func (r *UserRepository) GetRawByID(id model.UserUUID) (interface{}, error) {
	raw, err := r.db.First(model.UserType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *UserRepository) GetByID(id model.UserUUID) (*model.User, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.User), err
}

func (r *UserRepository) Update(user *model.User) error {
	_, err := r.GetByID(user.UUID)
	if err != nil {
		return err
	}
	return r.save(user)
}

func (r *UserRepository) Delete(id model.UserUUID, archivingTimestamp model.UnixTime, archivingHash int64) error {
	user, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if user.IsDeleted() {
		return model.ErrIsArchived
	}
	user.ArchivingTimestamp = archivingTimestamp
	user.ArchivingHash = archivingHash
	return r.Update(user)
}

func (r *UserRepository) List(tenantUUID model.TenantUUID, showArchived bool) ([]*model.User, error) {
	iter, err := r.db.Get(model.UserType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.User{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.User)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *UserRepository) ListIDs(tenantID model.TenantUUID, showArchived bool) ([]model.UserUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.UserUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *UserRepository) Iter(action func(*model.User) (bool, error)) error {
	iter, err := r.db.Get(model.UserType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.User)
		next, err := action(obj)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
}

func (r *UserRepository) Sync(_ string, data []byte) error {
	user := &model.User{}
	err := json.Unmarshal(data, user)
	if err != nil {
		return err
	}

	return r.save(user)
}

func (r *UserRepository) Restore(id model.UserUUID) (*model.User, error) {
	user, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user.ArchivingTimestamp == 0 {
		return nil, model.ErrIsNotArchived
	}
	user.ArchivingTimestamp = 0
	user.ArchivingHash = 0
	err = r.Update(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
