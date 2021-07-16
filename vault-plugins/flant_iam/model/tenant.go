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

func (r *TenantRepository) Delete(id TenantUUID) error {
	tenant, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(TenantType, tenant)
}

func (r *TenantRepository) List() ([]*Tenant, error) {
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
		list = append(list, obj)
	}
	return list, nil
}

func (r *TenantRepository) ListIDs() ([]TenantUUID, error) {
	objs, err := r.List()
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

func (r *TenantRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	tenant := &Tenant{}
	err := json.Unmarshal(data, tenant)
	if err != nil {
		return err
	}

	return r.save(tenant)
}
