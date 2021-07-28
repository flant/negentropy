package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	SourceTenantUUIDIndex      = TenantForeignPK // it is generated because tenant is the parent object for shares
	DestinationTenantUUIDIndex = "destination_tenant_uuid_index"
)

func IdentitySharingSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			IdentitySharingType: {
				Name: IdentitySharingType,
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

type IdentitySharing struct {
	UUID                  IdentitySharingUUID `json:"uuid"` // PK
	SourceTenantUUID      TenantUUID          `json:"source_tenant_uuid"`
	DestinationTenantUUID TenantUUID          `json:"destination_tenant_uuid"`

	Version string `json:"resource_version"`

	// Groups which to share with target tenant
	Groups []GroupUUID `json:"groups"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

const IdentitySharingType = "identity_sharing" // also, memdb schema name

func (u *IdentitySharing) ObjType() string {
	return IdentitySharingType
}

func (u *IdentitySharing) ObjId() string {
	return u.UUID
}

type IdentitySharingRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewIdentitySharingRepository(tx *io.MemoryStoreTxn) *IdentitySharingRepository {
	return &IdentitySharingRepository{db: tx}
}

func (r *IdentitySharingRepository) save(sh *IdentitySharing) error {
	return r.db.Insert(IdentitySharingType, sh)
}

func (r *IdentitySharingRepository) Create(sh *IdentitySharing) error {
	return r.save(sh)
}

func (r *IdentitySharingRepository) GetRawByID(id IdentitySharingUUID) (interface{}, error) {
	raw, err := r.db.First(IdentitySharingType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *IdentitySharingRepository) GetByID(id IdentitySharingUUID) (*IdentitySharing, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*IdentitySharing), err
}

func (r *IdentitySharingRepository) Update(sh *IdentitySharing) error {
	_, err := r.GetByID(sh.UUID)
	if err != nil {
		return err
	}
	return r.save(sh)
}

func (r *IdentitySharingRepository) Delete(id IdentitySharingUUID,
	archivingTimestamp UnixTime, archivingHash int64) error {
	sh, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if sh.ArchivingTimestamp != 0 {
		return ErrIsArchived
	}
	sh.ArchivingTimestamp = archivingTimestamp
	sh.ArchivingHash = archivingHash
	return r.Update(sh)
}

func (r *IdentitySharingRepository) List(tenantUUID TenantUUID, showArchived bool) ([]*IdentitySharing, error) {
	iter, err := r.db.Get(IdentitySharingType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*IdentitySharing{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*IdentitySharing)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *IdentitySharingRepository) ListIDs(tenantID TenantUUID, showArchived bool) ([]IdentitySharingUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]IdentitySharingUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *IdentitySharingRepository) Iter(action func(*IdentitySharing) (bool, error)) error {
	iter, err := r.db.Get(IdentitySharingType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*IdentitySharing)
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
	sh := &IdentitySharing{}
	err := json.Unmarshal(data, sh)
	if err != nil {
		return err
	}

	return r.save(sh)
}

func (r *IdentitySharingRepository) ListForDestinationTenant(tenantID TenantUUID) ([]*IdentitySharing, error) {
	iter, err := r.db.Get(IdentitySharingType, DestinationTenantUUIDIndex, tenantID)
	if err != nil {
		return nil, err
	}

	res := make([]*IdentitySharing, 0)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*IdentitySharing)
		res = append(res, u)
	}
	return res, nil
}
