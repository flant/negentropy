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

	Users           []UserUUID           `json:"-"`
	Groups          []GroupUUID          `json:"-"`
	ServiceAccounts []ServiceAccountUUID `json:"-"`
	Subjects        []SubjectNotation    `json:"subjects"`

	AnyProject bool          `json:"any_project"`
	Projects   []ProjectUUID `json:"projects"`

	Roles []BoundRole `json:"roles"`

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
	Scoep   RoleScope              `json:"scope"`
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

func (r *RoleBindingRepository) save(rb *RoleBinding) error {
	return r.db.Insert(RoleBindingType, rb)
}

func (r *RoleBindingRepository) fillSubjects(rb *RoleBinding) error {
	subj, err := NewSubjectsFetcher(r.db, rb.Subjects).Fetch()
	if err != nil {
		return err
	}
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users
	return nil
}

func (r *RoleBindingRepository) Create(rb *RoleBinding) error {
	// Validate
	if rb.Origin == "" {
		return ErrBadOrigin
	}
	if rb.Version != "" {
		return ErrBadVersion
	}
	rb.Version = NewResourceVersion()

	// Refill data
	if err := r.fillSubjects(rb); err != nil {
		return err
	}

	return r.save(rb)
}

func (r *RoleBindingRepository) Update(rb *RoleBinding) error {
	// Validate
	if rb.Origin == "" {
		return ErrBadOrigin
	}

	// Validate tenant relation
	if stored, err := r.GetByID(rb.UUID); err != nil {
		return err
	} else if stored.TenantUUID != rb.TenantUUID {
		return ErrNotFound
	}

	// Refill data
	if err := r.fillSubjects(rb); err != nil {
		return err
	}

	// Store
	return r.save(rb)
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

func (r *RoleBindingRepository) delete(id RoleBindingUUID) error {
	rb, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(RoleBindingType, rb)
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

func (r *RoleBindingRepository) List(tid TenantUUID) ([]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, TenantForeignPK, tid)
	if err != nil {
		return nil, err
	}

	list := []*RoleBinding{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		rb := raw.(*RoleBinding)
		list = append(list, rb)
	}
	return list, nil
}

func (r *RoleBindingRepository) DeleteByTenant(tid TenantUUID) error {
	_, err := r.db.DeleteAll(RoleBindingType, TenantForeignPK, tid)
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

func (r *RoleBindingRepository) UnsetExtension(origin ObjectOrigin, rbid RoleBindingUUID) error {
	obj, err := r.GetByID(rbid)
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
