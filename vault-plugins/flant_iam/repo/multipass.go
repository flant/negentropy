package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const (
	OwnerForeignPK = "owner_uuid"
)

func MultipassSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.MultipassType: {
				Name: model.MultipassType,
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
					OwnerForeignPK: {
						Name: OwnerForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "OwnerUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.MultipassType: {
				{OriginalDataTypeFieldName: "TenantUUID", RelatedDataType: model.TenantType, RelatedDataTypeFieldIndexName: PK},
			},
		},
	}
}

type MultipassRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewMultipassRepository(tx *io.MemoryStoreTxn) *MultipassRepository {
	return &MultipassRepository{db: tx}
}

func (r *MultipassRepository) save(multipass *model.Multipass) error {
	return r.db.Insert(model.MultipassType, multipass)
}

func (r *MultipassRepository) Create(multipass *model.Multipass) error {
	return r.save(multipass)
}

func (r *MultipassRepository) GetRawByID(id model.MultipassUUID) (interface{}, error) {
	raw, err := r.db.First(model.MultipassType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *MultipassRepository) GetByID(id model.MultipassUUID) (*model.Multipass, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Multipass), err
}

func (r *MultipassRepository) Update(multipass *model.Multipass) error {
	_, err := r.GetByID(multipass.UUID)
	if err != nil {
		return err
	}
	return r.save(multipass)
}

func (r *MultipassRepository) Delete(id model.MultipassUUID, archiveMark memdb.ArchiveMark) error {
	multipass, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if multipass.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(model.MultipassType, multipass, archiveMark)
}

func (r *MultipassRepository) List(ownerUUID model.OwnerUUID, showArchived bool) ([]*model.Multipass, error) {
	iter, err := r.db.Get(model.MultipassType, OwnerForeignPK, ownerUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.Multipass{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Multipass)
		if showArchived || obj.Timestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *MultipassRepository) ListIDs(ownerID model.OwnerUUID, showArchived bool) ([]model.MultipassUUID, error) {
	objs, err := r.List(ownerID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.MultipassUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *MultipassRepository) Iter(action func(*model.Multipass) (bool, error)) error {
	iter, err := r.db.Get(model.MultipassType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Multipass)
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

func (r *MultipassRepository) Sync(_ string, data []byte) error {
	multipass := &model.Multipass{}
	err := json.Unmarshal(data, multipass)
	if err != nil {
		return err
	}

	return r.save(multipass)
}
