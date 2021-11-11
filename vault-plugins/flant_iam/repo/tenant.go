package repo

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	TenantForeignPK = "tenant_uuid"
)

func TenantSchema() map[string]*memdb.TableSchema {
	return map[string]*memdb.TableSchema{
		model.TenantType: {
			Name: model.TenantType,
			Indexes: map[string]*memdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &memdb.UUIDFieldIndex{
						Field: "UUID",
					},
				},
				"identifier": {
					Name:   "identifier",
					Unique: true,
					Indexer: &memdb.StringFieldIndex{
						Field:     "Identifier",
						Lowercase: true,
					},
				},
				"version": {
					Name: "version",
					Indexer: &memdb.StringFieldIndex{
						Field: "Version",
					},
				},
			},
		},
	}
}

type TenantRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewTenantRepository(tx *io.MemoryStoreTxn) *TenantRepository {
	return &TenantRepository{db: tx}
}

func (r *TenantRepository) save(tenant *model.Tenant) error {
	return r.db.Insert(model.TenantType, tenant)
}

func (r *TenantRepository) Create(tenant *model.Tenant) error {
	return r.save(tenant)
}

func (r *TenantRepository) GetRawByID(id model.TenantUUID) (interface{}, error) {
	raw, err := r.db.First(model.TenantType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *TenantRepository) GetByID(id model.TenantUUID) (*model.Tenant, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Tenant), err
}

func (r *TenantRepository) Update(tenant *model.Tenant) error {
	_, err := r.GetByID(tenant.UUID)
	if err != nil {
		return err
	}
	return r.save(tenant)
}

func (r *TenantRepository) Delete(id model.TenantUUID, archivingTimestamp model.UnixTime, archivingHash int64) error {
	tenant, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if tenant.IsDeleted() {
		return model.ErrIsArchived
	}
	tenant.ArchivingTimestamp = archivingTimestamp
	tenant.ArchivingHash = archivingHash
	return r.Update(tenant)
}

func (r *TenantRepository) List(showArchived bool) ([]*model.Tenant, error) {
	iter, err := r.db.Get(model.TenantType, PK)
	if err != nil {
		return nil, err
	}

	list := []*model.Tenant{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Tenant)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *TenantRepository) ListIDs(showArchived bool) ([]model.TenantUUID, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.TenantUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *TenantRepository) Iter(action func(*model.Tenant) (bool, error)) error {
	iter, err := r.db.Get(model.TenantType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Tenant)
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

func (r *TenantRepository) Sync(_ string, data []byte) error {
	tenant := &model.Tenant{}
	err := json.Unmarshal(data, tenant)
	if err != nil {
		return err
	}

	return r.save(tenant)
}

func (r *TenantRepository) Restore(id model.TenantUUID) (*model.Tenant, error) {
	tenant, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if tenant.ArchivingTimestamp == 0 {
		return nil, model.ErrIsNotArchived
	}
	tenant.ArchivingTimestamp = 0
	tenant.ArchivingHash = 0
	err = r.Update(tenant)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}
