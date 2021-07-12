package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	TenantType      = "tenant" // also, memdb schema name
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

type TenantFeatureFlag struct {
	FeatureFlag           `json:",inline"`
	EnabledForNewProjects bool `json:"enabled_for_new"`
}

func (t *Tenant) ObjType() string {
	return TenantType
}

func (t *Tenant) ObjId() string {
	return t.UUID
}

type TenantRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics

}

func NewTenantRepository(tx *io.MemoryStoreTxn) *TenantRepository {
	return &TenantRepository{
		db: tx,
	}
}

func (r *TenantRepository) save(t *Tenant) error {
	return r.db.Insert(TenantType, t)
}

func (r *TenantRepository) Create(t *Tenant) error {
	t.Version = NewResourceVersion()
	return r.db.Insert(TenantType, t)
}

func (r *TenantRepository) GetByID(id TenantUUID) (*Tenant, error) {
	raw, err := r.db.First(TenantType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*Tenant), nil
}

func (r *TenantRepository) Update(updated *Tenant) error {
	stored, err := r.GetByID(updated.UUID)
	if err != nil {
		return err
	}

	// Validate

	if stored.Version != updated.Version {
		return ErrBadVersion
	}
	updated.Version = NewResourceVersion()

	// Update

	return r.db.Insert(TenantType, updated)
}

func (r *TenantRepository) Delete(id TenantUUID) error {
	err := r.deleteNestedObjects(
		id,
		NewUserRepository(r.db),
		NewProjectRepository(r.db),
		NewServiceAccountRepository(r.db),
		NewGroupRepository(r.db),
		NewRoleBindingRepository(r.db),
	)
	if err != nil {
		return err
	}

	return r.delete(id)
}

func (r *TenantRepository) delete(id string) error {
	tenant, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(TenantType, tenant)
}

func (r *TenantRepository) deleteNestedObjects(id string, repos ...SubTenantRepo) error {
	for _, r := range repos {
		err := r.DeleteByTenant(id)
		if err != nil {
			return err
		}
	}
	return nil
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
		t := raw.(*Tenant)
		list = append(list, t)
	}
	return list, nil
}

func (r *TenantRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.delete(objID)
	}

	tenant := &Tenant{}
	err := json.Unmarshal(data, tenant)
	if err != nil {
		return err
	}

	return r.save(tenant)
}

type SubTenantRepo interface {
	DeleteByTenant(string) error
}
