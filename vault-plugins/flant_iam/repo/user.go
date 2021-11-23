package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func UserSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.UserType: {
				Name: model.UserType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					"version": {
						Name: "version",
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Version",
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Identifier",
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.UserType: {{OriginalDataTypeFieldName: "TenantUUID", RelatedDataType: model.TenantType, RelatedDataTypeFieldIndexName: PK}},
		},
		CascadeDeletes: map[string][]memdb.Relation{
			model.UserType: {
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.GroupType, RelatedDataTypeFieldIndexName: UserInGroupIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingType, RelatedDataTypeFieldIndexName: UserInRoleBindingIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingApprovalType, RelatedDataTypeFieldIndexName: UserInRoleBindingApprovalIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.MultipassType, RelatedDataTypeFieldIndexName: OwnerForeignPK},
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
		return nil, consts.ErrNotFound
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

func (r *UserRepository) CascadeDelete(id model.UserUUID, archiveMark memdb.ArchiveMark) error {
	user, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if user.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.CascadeArchive(model.UserType, user, archiveMark)
}

func (r *UserRepository) CleanChildrenSliceIndexes(id model.UserUUID) error {
	user, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.CleanChildrenSliceIndexes(model.UserType, user)
}

func (r *UserRepository) CascadeRestore(id model.UserUUID) (*model.User, error) {
	user, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !user.Archived() {
		return nil, consts.ErrIsNotArchived
	}
	err = r.db.CascadeRestore(model.UserType, user)
	if err != nil {
		return nil, err
	}
	return user, nil
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
		if showArchived || obj.Timestamp == 0 {
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
	if user.Timestamp == 0 {
		return nil, consts.ErrIsNotArchived
	}
	user.Timestamp = 0
	user.Hash = 0
	err = r.Update(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
