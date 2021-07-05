package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	GroupType = "group" // also, memdb schema name

)

type GroupObjectType string

func GroupSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			GroupType: {
				Name: GroupType,
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

/*
identifier – уникален в рамках тенанта для каждого builtin_type_name
Пользователи
Группы
Сервисные аккаунты

uuid
tenant_uuid
identifier
full_identifier:

users
service_accounts
groups
resource_version
*/
type Group struct {
	UUID            string   `json:"uuid"` // PK
	TenantUUID      string   `json:"tenant_uuid"`
	Version         string   `json:"resource_version"`
	BuiltinType     string   `json:"-"`
	Identifier      string   `json:"identifier"`
	FullIdentifier  string   `json:"full_identifier"`
	Users           []string `json:"users"`
	Groups          []string `json:"groups"`
	ServiceAccounts []string `json:"service_accounts"`
}

func (u *Group) ObjType() string {
	return GroupType
}

func (u *Group) ObjId() string {
	return u.UUID
}

func (u *Group) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(u)
}

func (u *Group) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, u)
	return err
}

// generic: <identifier>@group.<tenant_identifier>
// builtin: <identifier>@<builtin_group_type>.group.<tenant_identifier>
func CalcGroupFullIdentifier(g *Group, tenant *Tenant) string {
	name := g.Identifier
	domain := "group." + tenant.Identifier
	if g.BuiltinType != "" {
		domain = g.BuiltinType + "." + domain
	}
	return name + "@" + domain
}

type GroupRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewGroupRepository(tx *io.MemoryStoreTxn) *GroupRepository {
	return &GroupRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *GroupRepository) Create(group *Group) error {
	tenant, err := r.tenantRepo.GetById(group.TenantUUID)
	if err != nil {
		return err
	}

	if group.Version != "" {
		return ErrVersionMismatch
	}
	group.Version = NewResourceVersion()
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	err = r.db.Insert(GroupType, group)
	if err != nil {
		return err
	}
	return nil
}

func (r *GroupRepository) GetById(id string) (*Group, error) {
	raw, err := r.db.First(GroupType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	group := raw.(*Group)
	return group, nil
}

func (r *GroupRepository) Update(group *Group) error {
	stored, err := r.GetById(group.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != group.TenantUUID {
		return ErrNotFound
	}
	if stored.Version != group.Version {
		return ErrVersionMismatch
	}
	group.Version = NewResourceVersion()

	// Update

	tenant, err := r.tenantRepo.GetById(group.TenantUUID)
	if err != nil {
		return err
	}
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	err = r.db.Insert(GroupType, group)
	if err != nil {
		return err
	}

	return nil
}

/*
TODO Clean from everywhere:
	* other groups
	* role_bindings
	* approvals
	* identity_sharings
*/
func (r *GroupRepository) Delete(id string) error {
	group, err := r.GetById(id)
	if err != nil {
		return err
	}

	return r.db.Delete(GroupType, group)
}

func (r *GroupRepository) List(tenantID string) ([]string, error) {
	iter, err := r.db.Get(GroupType, TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		u := raw.(*Group)
		ids = append(ids, u.UUID)
	}
	return ids, nil
}

func (r *GroupRepository) DeleteByTenant(tenantUUID string) error {
	_, err := r.db.DeleteAll(GroupType, TenantForeignPK, tenantUUID)
	return err
}
