package model

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	IncludedRolesIndex = "IncludedRolesIndex"
	RoleScopeIndex     = "scope"
	ArchivedAtIndex    = "archived_at_index"
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
					ArchivedAtIndex: {
						Name:         ArchivedAtIndex,
						AllowMissing: false,
						Unique:       false,
						Indexer: &memdb.IntFieldIndex{
							Field: "ArchivingTimestamp",
						},
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

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
	// FIXME add version?
}

type IncludedRole struct {
	Name            RoleName `json:"name"`
	OptionsTemplate string   `json:"options_template"`
}

const RoleType = "role" // also, memdb schema name

func (r *Role) isDeleted() bool {
	return r.ArchivingTimestamp != 0
}

func (r *Role) ObjType() string {
	return RoleType
}

func (r *Role) ObjId() string {
	return r.Name
}

type RoleRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleRepository(tx *io.MemoryStoreTxn) *RoleRepository {
	return &RoleRepository{db: tx}
}

func (r *RoleRepository) save(role *Role) error {
	return r.db.Insert(RoleType, role)
}

func (r *RoleRepository) Create(role *Role) error {
	return r.save(role)
}

func (r *RoleRepository) GetRawByID(roleID RoleName) (interface{}, error) {
	raw, err := r.db.First(RoleType, PK, roleID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *RoleRepository) GetByID(roleID RoleName) (*Role, error) {
	raw, err := r.GetRawByID(roleID)
	if raw == nil {
		return nil, err
	}
	return raw.(*Role), err
}

func (r *RoleRepository) Update(role *Role) error {
	_, err := r.GetByID(role.Name)
	if err != nil {
		return err
	}
	return r.save(role)
}

func (r *RoleRepository) List(showArchived bool) ([]*Role, error) {
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
		obj := raw.(*Role)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *RoleRepository) ListIDs(showArchived bool) ([]RoleName, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]RoleName, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *RoleRepository) Delete(roleID RoleName, archivingTimestamp UnixTime, archivingHash int64) error {
	role, err := r.GetByID(roleID)
	if err != nil {
		return err
	}
	if role.isDeleted() {
		return ErrIsArchived
	}
	role.ArchivingTimestamp = archivingTimestamp
	role.ArchivingHash = archivingHash
	return r.Update(role)
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
		obj := raw.(*Role)
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

func (r *RoleRepository) Sync(objID string, data []byte) error {
	role := &Role{}
	err := json.Unmarshal(data, role)
	if err != nil {
		return err
	}

	return r.save(role)
}

func (r *RoleRepository) FindDirectIncludingRoles(roleID RoleName) (map[RoleName]struct{}, error) {
	iter, err := r.db.Get(RoleType, IncludedRolesIndex, roleID)
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
		if role.ArchivingTimestamp == 0 {
			ids[role.Name] = struct{}{}
		}
	}
	return ids, nil
}

func (r *RoleRepository) FindAllIncludingRoles(roleID RoleName) (map[RoleName]struct{}, error) {
	result := map[RoleName]struct{}{} // empty map
	role, err := r.GetByID(roleID)
	if err != nil {
		return nil, err
	}
	if role.isDeleted() {
		return result, nil
	}
	currentSet := map[RoleName]struct{}{roleID: {}}
	for len(currentSet) != 0 {
		nextSet := map[RoleName]struct{}{}
		for currentRole := range currentSet {
			candidates, err := r.FindDirectIncludingRoles(currentRole)
			if err != nil {
				return nil, err
			}
			for candidate := range candidates {
				if _, found := result[candidate]; !found && candidate != roleID {
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
