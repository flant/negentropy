package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func ServiceAccountPasswordSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.ServiceAccountPasswordType: {
				Name: model.ServiceAccountPasswordType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					OwnerForeignPK: {
						Name: OwnerForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "OwnerUUID",
							Lowercase: true,
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.ServiceAccountPasswordType: {
				{OriginalDataTypeFieldName: "TenantUUID", RelatedDataType: model.TenantType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "OwnerUUID", RelatedDataType: model.ServiceAccountType, RelatedDataTypeFieldIndexName: PK},
			},
		},
	}
}

type ServiceAccountPasswordRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewServiceAccountPasswordRepository(tx *io.MemoryStoreTxn) *ServiceAccountPasswordRepository {
	return &ServiceAccountPasswordRepository{db: tx}
}

func (r *ServiceAccountPasswordRepository) save(sap *model.ServiceAccountPassword) error {
	return r.db.Insert(model.ServiceAccountPasswordType, sap)
}

func (r *ServiceAccountPasswordRepository) Create(sap *model.ServiceAccountPassword) error {
	return r.save(sap)
}

func (r *ServiceAccountPasswordRepository) GetRawByID(id model.ServiceAccountPasswordUUID) (interface{}, error) {
	raw, err := r.db.First(model.ServiceAccountPasswordType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *ServiceAccountPasswordRepository) GetByID(id model.ServiceAccountPasswordUUID) (*model.ServiceAccountPassword, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.ServiceAccountPassword), err
}

func (r *ServiceAccountPasswordRepository) Update(sap *model.ServiceAccountPassword) error {
	_, err := r.GetByID(sap.UUID)
	if err != nil {
		return err
	}
	return r.save(sap)
}

func (r *ServiceAccountPasswordRepository) Delete(id model.ServiceAccountPasswordUUID,
	archiveMark memdb.ArchiveMark) error {
	sap, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if sap.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(model.ServiceAccountType, sap, archiveMark)
}

func (r *ServiceAccountPasswordRepository) List(ownerUUID model.OwnerUUID,
	showArchived bool) ([]*model.ServiceAccountPassword, error) {
	iter, err := r.db.Get(model.ServiceAccountPasswordType, OwnerForeignPK, ownerUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.ServiceAccountPassword{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.ServiceAccountPassword)
		if showArchived || obj.Timestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *ServiceAccountPasswordRepository) ListIDs(ownerID model.OwnerUUID,
	showArchived bool) ([]model.ServiceAccountPasswordUUID, error) {
	objs, err := r.List(ownerID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.ServiceAccountPasswordUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ServiceAccountPasswordRepository) Iter(action func(*model.ServiceAccountPassword) (bool, error)) error {
	iter, err := r.db.Get(model.ServiceAccountPasswordType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.ServiceAccountPassword)
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

func (r *ServiceAccountPasswordRepository) Sync(_ string, data []byte) error {
	sap := &model.ServiceAccountPassword{}
	err := json.Unmarshal(data, sap)
	if err != nil {
		return err
	}

	return r.save(sap)
}
