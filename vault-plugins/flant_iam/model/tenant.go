package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	TenantForeignPK = "tenant_uuid"
)

func TenantSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			TenantType: {
				Name: TenantType,
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
		},
	}
}

type Tenant struct {
	UUID       TenantUUID `json:"uuid"` // PK
	Version    string     `json:"resource_version"`
	Identifier string     `json:"identifier"`

	FeatureFlags []TenantFeatureFlag `json:"feature_flags"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

const TenantType = "tenant" // also, memdb schema name

func (u *Tenant) ObjType() string {
	return TenantType
}

func (u *Tenant) ObjId() string {
	return u.UUID
}

type TenantRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewTenantRepository(tx *io.MemoryStoreTxn) *TenantRepository {
	return &TenantRepository{db: tx}
}

func (r *TenantRepository) save(tenant *Tenant) error {
	return r.db.Insert(TenantType, tenant)
}

func (r *TenantRepository) Create(tenant *Tenant) error {
	return r.save(tenant)
}

func (r *TenantRepository) GetRawByID(id TenantUUID) (interface{}, error) {
	raw, err := r.db.First(TenantType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *TenantRepository) GetByID(id TenantUUID) (*Tenant, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*Tenant), err
}

func (r *TenantRepository) Update(tenant *Tenant) error {
	_, err := r.GetByID(tenant.UUID)
	if err != nil {
		return err
	}
	return r.save(tenant)
}

func (r *TenantRepository) Delete(id TenantUUID, archivingTimestamp UnixTime, archivingHash int64) error {
	tenant, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if tenant.ArchivingTimestamp != 0 {
		return ErrIsArchived
	}
	tenant.ArchivingTimestamp = archivingTimestamp
	tenant.ArchivingHash = archivingHash
	return r.Update(tenant)
}

func (r *TenantRepository) List(showArchived bool) ([]*Tenant, error) {
	iter, err := r.db.Get(TenantType, PK)
	if err != nil {
		return nil, err
	}

	list := []*Tenant{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Tenant)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *TenantRepository) ListIDs(showArchived bool) ([]TenantUUID, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]TenantUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *TenantRepository) Iter(action func(*Tenant) (bool, error)) error {
	iter, err := r.db.Get(TenantType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Tenant)
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
	tenant := &Tenant{}
	err := json.Unmarshal(data, tenant)
	if err != nil {
		return err
	}

	return r.save(tenant)
}

func (r *TenantRepository) Restore(id TenantUUID) (*Tenant, error) {
	tenant, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if tenant.ArchivingTimestamp == 0 {
		return nil, ErrIsNotArchived
	}
	tenant.ArchivingTimestamp = 0
	tenant.ArchivingHash = 0
	err = r.Update(tenant)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}
