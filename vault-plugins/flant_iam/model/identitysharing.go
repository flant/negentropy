package model

import (
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	IdentitySharingType        = "identity_sharing"
	SourceTenantUUIDIndex      = "source_tenant_uuid_index"
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
}

func (t *IdentitySharing) ObjType() string {
	return IdentitySharingType
}

func (t *IdentitySharing) ObjId() string {
	return t.UUID
}

type IdentitySharingRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics

	tenantRepo *TenantRepository
}

func NewIdentitySharingRepository(tx *io.MemoryStoreTxn) *IdentitySharingRepository {
	return &IdentitySharingRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *IdentitySharingRepository) save(ra *IdentitySharing) error {
	return r.db.Insert(IdentitySharingType, ra)
}

func (r *IdentitySharingRepository) delete(id IdentitySharingUUID) error {
	ra, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(IdentitySharingType, ra)
}

func (r *IdentitySharingRepository) GetByID(id IdentitySharingUUID) (*IdentitySharing, error) {
	raw, err := r.db.First(IdentitySharingType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	ra := raw.(*IdentitySharing)
	return ra, nil
}

func (r *IdentitySharingRepository) Create(is *IdentitySharing) error {
	_, err := r.tenantRepo.GetByID(is.SourceTenantUUID)
	if err != nil {
		return err
	}
	_, err = r.tenantRepo.GetByID(is.DestinationTenantUUID)
	if err != nil {
		return err
	}

	is.Version = NewResourceVersion()
	return r.save(is)
}

func (r *IdentitySharingRepository) Update(ra *IdentitySharing) error {
	ra.Version = NewResourceVersion()

	// Update
	return r.save(ra)
}

func (r *IdentitySharingRepository) Delete(id IdentitySharingUUID) error {
	return r.delete(id)
}

func (r *IdentitySharingRepository) Iter(action func(is *IdentitySharing) (bool, error)) error {
	iter, err := r.db.Get(IdentitySharingType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*IdentitySharing)
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

func (r *IdentitySharingRepository) List(tenantID TenantUUID) ([]*IdentitySharing, error) {
	iter, err := r.db.Get(IdentitySharingType, SourceTenantUUIDIndex, tenantID)
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
