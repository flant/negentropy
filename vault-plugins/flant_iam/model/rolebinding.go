package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	RoleBindingType = "role_binding" // also, memdb schema name

)

type RoleBindingObjectType string

func RoleBindingSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			RoleBindingType: {
				Name: RoleBindingType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
				},
			},
		},
	}
}

type RoleBinding struct {
	UUID       RoleBindingUUID `json:"uuid"` // PK
	TenantUUID TenantUUID      `json:"tenant_uuid"`
	Version    string          `json:"resource_version"`

	ValidTill  int64 `json:"valid_till"`
	RequireMFA bool  `json:"require_mfa"`

	Users           []UserUUID           `json:"users"`
	Groups          []GroupUUID          `json:"groups"`
	ServiceAccounts []ServiceAccountUUID `json:"service_accounts"`

	AnyProject bool          `json:"any_project"`
	Projects   []ProjectUUID `json:"projects"`

	Roles []BoundRole `json:"-"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"extensions"`
}

func (u *RoleBinding) ObjType() string {
	return RoleBindingType
}

func (u *RoleBinding) ObjId() string {
	return u.UUID
}

type BoundRole struct {
	Name    RoleName               `json:"name"`
	Version string                 `json:"resource_version"`
	Options map[string]interface{} `json:"options"`
}

type RoleBindingRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewRoleBindingRepository(tx *io.MemoryStoreTxn) *RoleBindingRepository {
	return &RoleBindingRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *RoleBindingRepository) save(roleBinding *RoleBinding) error {
	return r.db.Insert(RoleBindingType, roleBinding)
}

func (r *RoleBindingRepository) Create(roleBinding *RoleBinding) error {
	_, err := r.tenantRepo.GetByID(roleBinding.TenantUUID)
	if err != nil {
		return err
	}

	if roleBinding.Version != "" {
		return ErrBadVersion
	}
	if roleBinding.Origin == "" {
		return ErrBadOrigin
	}
	roleBinding.Version = NewResourceVersion()

	return r.save(roleBinding)
}

func (r *RoleBindingRepository) GetByID(id RoleBindingUUID) (*RoleBinding, error) {
	raw, err := r.db.First(RoleBindingType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	roleBinding := raw.(*RoleBinding)
	return roleBinding, nil
}

func (r *RoleBindingRepository) Update(roleBinding *RoleBinding) error {
	stored, err := r.GetByID(roleBinding.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != roleBinding.TenantUUID {
		return ErrNotFound
	}
	if roleBinding.Origin != stored.Origin {
		return ErrBadOrigin
	}
	if stored.Version != roleBinding.Version {
		return ErrBadVersion
	}
	roleBinding.Version = NewResourceVersion()

	// Update
	return r.save(roleBinding)
}

func (r *RoleBindingRepository) delete(id RoleBindingUUID) error {
	roleBinding, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(RoleBindingType, roleBinding)
}

func (r *RoleBindingRepository) Delete(origin ObjectOrigin, id RoleBindingUUID) error {
	roleBinding, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if roleBinding.Origin != origin {
		return ErrBadOrigin
	}
	return r.delete(id)
}

func (r *RoleBindingRepository) List(tenantID TenantUUID) ([]RoleBindingUUID, error) {
	iter, err := r.db.Get(RoleBindingType, TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []RoleBindingUUID{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*RoleBinding)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}

func (r *RoleBindingRepository) DeleteByTenant(tenantUUID TenantUUID) error {
	_, err := r.db.DeleteAll(RoleBindingType, TenantForeignPK, tenantUUID)
	return err
}

func (r *RoleBindingRepository) SetExtension(ext *Extension) error {
	obj, err := r.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[ObjectOrigin]*Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return r.save(obj)
}

func (r *RoleBindingRepository) UnsetExtension(origin ObjectOrigin, uuid RoleBindingUUID) error {
	obj, err := r.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return r.save(obj)
}

func (r *RoleBindingRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.delete(objID)
	}

	rb := &RoleBinding{}
	err := json.Unmarshal(data, rb)
	if err != nil {
		return err
	}

	return r.save(rb)
}
