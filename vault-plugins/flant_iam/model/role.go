package model

import (
	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	RoleType = "role" // also, memdb schema name
)

func RoleSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			RoleType: {
				Name: RoleType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
					"type": {
						Name: "type",
						Indexer: &memdb.StringFieldIndex{
							Field: "Type",
						},
					},
				},
			},
		},
	}
}

type GroupScope string

const (
	GroupScopeTenant  GroupScope = "tenant"
	GroupScopeProject GroupScope = "project"
)

type Role struct {
	Name RoleName   `json:"name"`
	Type GroupScope `json:"type"`

	Description   string `json:"description"`
	OptionsSchema string `json:"options_schema"`

	RequireOneOfFeatureFlags []FeatureFlagName `json:"require_one_of_feature_flags"`
	IncludedRoles            []IncludedRole    `json:"included_roles"`
}

type IncludedRole struct {
	Name            RoleName `json:"name"`
	OptionsTemplate string   `json:"options_template"`
}

func (t *Role) ObjType() string {
	return RoleType
}

func (t *Role) ObjId() string {
	return t.Name
}

func (t *Role) Marshal(_ bool) ([]byte, error) {
	return jsonutil.EncodeJSON(t)
}

func (t *Role) Unmarshal(data []byte) error {
	err := jsonutil.DecodeJSON(data, t)
	return err
}

type RoleRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics

}

func NewRoleRepository(tx *io.MemoryStoreTxn) *RoleRepository {
	return &RoleRepository{
		db: tx,
	}
}

func (r *RoleRepository) Create(t *Role) error {
	return r.db.Insert(RoleType, t)
}

func (r *RoleRepository) Get(name RoleName) (*Role, error) {
	raw, err := r.db.First(RoleType, PK, name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*Role), nil
}

func (r *RoleRepository) Update(updated *Role) error {
	stored, err := r.Get(updated.Name)
	if err != nil {
		return err
	}

	updated.Type = stored.Type // type cannot be changed

	// TODO validate feature flags: role must not become unaccessable in the scope where it is used
	// TODO forbid backwards-incompatible changes of the options schema

	return r.db.Insert(RoleType, updated)
}

func (r *RoleRepository) Delete(name RoleName) error {
	role, err := r.Get(name)
	if err != nil {
		return err
	}

	// TODO before the deletion, check it is not used in
	//      * role_biondings,
	//      * approvals,
	//      * tokens,
	//      * service account passwords

	return r.db.Delete(RoleType, role)
}

func (r *RoleRepository) List() ([]RoleName, error) {
	iter, err := r.db.Get(RoleType, PK)
	if err != nil {
		return nil, err
	}

	names := []RoleName{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*Role)
		names = append(names, t.Name)
	}
	return names, nil
}
