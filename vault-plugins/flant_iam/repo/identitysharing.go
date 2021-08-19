package repo

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	SourceTenantUUIDIndex      = TenantForeignPK // it is generated because tenant is the parent object for shares
	DestinationTenantUUIDIndex = "destination_tenant_uuid_index"
)

func IdentitySharingSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			model.IdentitySharingType: {
				Name: model.IdentitySharingType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					SourceTenantUUIDIndex: {
						Name: SourceTenantUUIDIndex,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "SourceTenantUUID",
						},
					},
					DestinationTenantUUIDIndex: {
						Name: DestinationTenantUUIDIndex,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "DestinationTenantUUID",
						},
					},
				},
			},
		},
	}
}

type IdentitySharingRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewIdentitySharingRepository(tx *io.MemoryStoreTxn) *IdentitySharingRepository {
	return &IdentitySharingRepository{db: tx}
}

func (r *IdentitySharingRepository) save(sh *model.IdentitySharing) error {
	return r.db.Insert(model.IdentitySharingType, sh)
}

func (r *IdentitySharingRepository) Create(sh *model.IdentitySharing) error {
	return r.save(sh)
}

func (r *IdentitySharingRepository) GetRawByID(id model.IdentitySharingUUID) (interface{}, error) {
	raw, err := r.db.First(model.IdentitySharingType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *IdentitySharingRepository) GetByID(id model.IdentitySharingUUID) (*model.IdentitySharing, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.IdentitySharing), err
}

func (r *IdentitySharingRepository) Update(sh *model.IdentitySharing) error {
	_, err := r.GetByID(sh.UUID)
	if err != nil {
		return err
	}
	return r.save(sh)
}

func (r *IdentitySharingRepository) Delete(id model.IdentitySharingUUID,
	archivingTimestamp model.UnixTime, archivingHash int64) error {
	sh, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if sh.IsDeleted() {
		return model.ErrIsArchived
	}
	sh.ArchivingTimestamp = archivingTimestamp
	sh.ArchivingHash = archivingHash
	return r.Update(sh)
}

func (r *IdentitySharingRepository) List(tenantUUID model.TenantUUID, showArchived bool) ([]*model.IdentitySharing, error) {
	iter, err := r.db.Get(model.IdentitySharingType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.IdentitySharing{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.IdentitySharing)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *IdentitySharingRepository) ListIDs(tenantID model.TenantUUID, showArchived bool) ([]model.IdentitySharingUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.IdentitySharingUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *IdentitySharingRepository) Iter(action func(*model.IdentitySharing) (bool, error)) error {
	iter, err := r.db.Get(model.IdentitySharingType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.IdentitySharing)
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

func (r *IdentitySharingRepository) Sync(_ string, data []byte) error {
	sh := &model.IdentitySharing{}
	err := json.Unmarshal(data, sh)
	if err != nil {
		return err
	}

	return r.save(sh)
}

func (r *IdentitySharingRepository) ListForDestinationTenant(tenantID model.TenantUUID) ([]*model.IdentitySharing, error) {
	iter, err := r.db.Get(model.IdentitySharingType, DestinationTenantUUIDIndex, tenantID)
	if err != nil {
		return nil, err
	}

	res := make([]*model.IdentitySharing, 0)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*model.IdentitySharing)
		res = append(res, u)
	}
	return res, nil
}
