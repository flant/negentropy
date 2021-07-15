package model

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	RoleType           = "role" // also, memdb schema name
	IncludedRolesIndex = "IncludedRolesIndex"
	RoleScopeIndex     = "scope"
)

type RoleScope string

const (
	RoleScopeTenant  RoleScope = "tenant"
	RoleScopeProject RoleScope = "project"
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
					RoleScopeIndex: {
						Name: RoleScopeIndex,
						Indexer: &memdb.StringFieldIndex{
							Field: "Scope",
						},
					},
					IncludedRolesIndex: {
						Name:         IncludedRolesIndex,
						AllowMissing: true,
						Unique:       false,
						Indexer:      includedRolesIndexer{},
					},
				},
			},
		},
	}
}

type Role struct {
	Name  RoleName  `json:"name"`
	Scope RoleScope `json:"scope"`

	Description   string `json:"description"`
	OptionsSchema string `json:"options_schema"`

	RequireOneOfFeatureFlags []FeatureFlagName `json:"require_one_of_feature_flags"`
	IncludedRoles            []IncludedRole    `json:"included_roles"`

	// FIXME add version?
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

type RoleRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleRepository(tx *io.MemoryStoreTxn) *RoleRepository {
	return &RoleRepository{
		db: tx,
	}
}

func (r *RoleRepository) save(role *Role) error {
	// TODO Validate exising name
	return r.db.Insert(RoleType, role)
}

func (r *RoleRepository) Create(role *Role) error {
	return r.save(role)
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

func (r *RoleRepository) Update(role *Role) error {
	_, err := r.Get(role.Name)
	if err != nil {
		return err
	}
	return r.save(role)
}

func (r *RoleRepository) Delete(name RoleName) error {
	role, err := r.Get(name)
	if err != nil {
		return err
	}
	return r.db.Delete(RoleType, role)
}

func (r *RoleRepository) List() ([]*Role, error) {
	iter, err := r.db.Get(RoleType, PK)
	if err != nil {
		return nil, err
	}

	list := []*Role{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		role := raw.(*Role)
		list = append(list, role)
	}
	return list, nil
}

func (r *RoleRepository) Iter(action func(*Role) (bool, error)) error {
	iter, err := r.db.Get(RoleType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*Role)
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

func (r *RoleRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	role := &Role{}
	err := json.Unmarshal(data, role)
	if err != nil {
		return err
	}

	return r.save(role)
}

func (r *RoleRepository) FindDirectIncludingRoles(role RoleName) (map[RoleName]struct{}, error) {
	iter, err := r.db.Get(RoleType, IncludedRolesIndex, role)
	if err != nil {
		return nil, err
	}
	ids := map[RoleName]struct{}{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		role := raw.(*Role)
		ids[role.Name] = struct{}{}
	}
	return ids, nil
}

func (r *RoleRepository) FindAllIncludingRoles(role RoleName) (map[RoleName]struct{}, error) {
	result := map[RoleName]struct{}{} // empty map
	currentSet := map[RoleName]struct{}{role: {}}
	for len(currentSet) != 0 {
		nextSet := map[RoleName]struct{}{}
		for currentRole := range currentSet {
			candidates, err := r.FindDirectIncludingRoles(currentRole)
			if err != nil {
				return nil, err
			}
			for candidate := range candidates {
				if _, found := result[candidate]; !found && candidate != role {
					result[candidate] = struct{}{}
					nextSet[candidate] = struct{}{}
				}
			}
		}
		currentSet = nextSet
	}
	return result, nil
}

type includedRolesIndexer struct{}

func (_ includedRolesIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, ErrNeedSingleArgument
	}
	arg, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be a string: %#v", args[0])
	}
	// Add the null character as a terminator
	arg += "\x00"
	return []byte(arg), nil
}

func (_ includedRolesIndexer) FromObject(raw interface{}) (bool, [][]byte, error) {
	role, ok := raw.(*Role)
	if !ok {
		return false, nil, fmt.Errorf("format error: need Role type, actual passed %#v", raw)
	}
	result := [][]byte{}
	for i := range role.IncludedRoles {
		result = append(result, []byte(role.IncludedRoles[i].Name+"\x00"))
	}
	if len(result) == 0 {
		return false, nil, nil
	}
	return true, result, nil
}
