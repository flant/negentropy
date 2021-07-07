package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"

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
	UUID        string `json:"uuid"` // PK
	TenantUUID  string `json:"tenant_uuid"`
	Version     string `json:"resource_version"`
	BuiltinType string `json:"-"`

	ValidTill  int64 `json:"valid_till"`
	RequireMFA bool  `json:"require_mfa"`

	Users           []string `json:"users"`
	Groups          []string `json:"groups"`
	ServiceAccounts []string `json:"service_accounts"`

	Roles                    []BoundRole               `json:"-"`
	MaterializedRoles        []MaterializedRole        `json:"-"`
	MaterializedProjectRoles []MaterializedProjectRole `json:"-"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"extensions"`
}

func (u *RoleBinding) ObjType() string {
	return RoleBindingType
}

func (u *RoleBinding) ObjId() string {
	return u.UUID
}

func (u RoleBinding) Marshal(includeSensitive bool) ([]byte, error) {
	obj := u
	if !includeSensitive {
		u := OmitSensitive(u).(RoleBinding)
		obj = u
	}
	return jsonutil.EncodeJSON(obj)
}

func (u *RoleBinding) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, u)
	return err
}

type BoundRole struct {
	Name       string                 `json:"name"`
	Version    string                 `json:"resource_version"`
	AnyProject bool                   `json:"any_project"`
	Projects   []string               `json:"projects"`
	Options    map[string]interface{} `json:"options"`
}

type MaterializedRole struct {
	Name    string                 `json:"name"`
	Options map[string]interface{} `json:"options"`
}

type MaterializedProjectRole struct {
	Project string `json:"project"`
	Name    string `json:"name"`
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

func (r *RoleBindingRepository) GetByID(id string) (*RoleBinding, error) {
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

func (r *RoleBindingRepository) delete(id string) error {
	roleBinding, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(RoleBindingType, roleBinding)
}

func (r *RoleBindingRepository) Delete(origin ObjectOrigin, id string) error {
	roleBinding, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if roleBinding.Origin != origin {
		return ErrBadOrigin
	}
	return r.delete(id)
}

func (r *RoleBindingRepository) List(tenantID string) ([]string, error) {
	iter, err := r.db.Get(RoleBindingType, TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
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

func (r *RoleBindingRepository) DeleteByTenant(tenantUUID string) error {
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

func (r *RoleBindingRepository) UnsetExtension(origin ObjectOrigin, uuid string) error {
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
