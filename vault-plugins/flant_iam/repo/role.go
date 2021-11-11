package repo

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	IncludedRolesIndex = "included_roles_index"
	RoleScopeIndex     = "scope"
	ArchivedAtIndex    = "archived_at_index"
)

func RoleSchema() map[string]*memdb.TableSchema {
	return map[string]*memdb.TableSchema{
		model.RoleType: {
			Name: model.RoleType,
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
	}
}

type RoleRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleRepository(tx *io.MemoryStoreTxn) *RoleRepository {
	return &RoleRepository{db: tx}
}

func (r *RoleRepository) save(role *model.Role) error {
	return r.db.Insert(model.RoleType, role)
}

func (r *RoleRepository) Create(role *model.Role) error {
	return r.save(role)
}

func (r *RoleRepository) GetRawByID(roleID model.RoleName) (interface{}, error) {
	raw, err := r.db.First(model.RoleType, PK, roleID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *RoleRepository) GetByID(roleID model.RoleName) (*model.Role, error) {
	raw, err := r.GetRawByID(roleID)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Role), err
}

func (r *RoleRepository) Update(role *model.Role) error {
	_, err := r.GetByID(role.Name)
	if err != nil {
		return err
	}
	return r.save(role)
}

func (r *RoleRepository) List(showArchived bool) ([]*model.Role, error) {
	iter, err := r.db.Get(model.RoleType, PK)
	if err != nil {
		return nil, err
	}

	list := []*model.Role{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Role)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *RoleRepository) ListIDs(showArchived bool) ([]model.RoleName, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.RoleName, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *RoleRepository) Delete(roleID model.RoleName, archivingTimestamp model.UnixTime, archivingHash int64) error {
	role, err := r.GetByID(roleID)
	if err != nil {
		return err
	}
	if role.IsDeleted() {
		return model.ErrIsArchived
	}
	role.ArchivingTimestamp = archivingTimestamp
	role.ArchivingHash = archivingHash
	return r.Update(role)
}

func (r *RoleRepository) Iter(action func(*model.Role) (bool, error)) error {
	iter, err := r.db.Get(model.RoleType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Role)
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
	role := &model.Role{}
	err := json.Unmarshal(data, role)
	if err != nil {
		return err
	}

	return r.save(role)
}

func (r *RoleRepository) FindDirectIncludingRoles(roleID model.RoleName) (map[model.RoleName]struct{}, error) {
	iter, err := r.db.Get(model.RoleType, IncludedRolesIndex, roleID)
	if err != nil {
		return nil, err
	}
	ids := map[model.RoleName]struct{}{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		role := raw.(*model.Role)
		if role.ArchivingTimestamp == 0 {
			ids[role.Name] = struct{}{}
		}
	}
	return ids, nil
}

func (r *RoleRepository) FindAllIncludingRoles(roleID model.RoleName) (map[model.RoleName]struct{}, error) {
	result := map[model.RoleName]struct{}{} // empty map
	role, err := r.GetByID(roleID)
	if err != nil {
		return nil, err
	}
	if role.IsDeleted() {
		return result, nil
	}
	currentSet := map[model.RoleName]struct{}{roleID: {}}
	for len(currentSet) != 0 {
		nextSet := map[model.RoleName]struct{}{}
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
		return nil, model.ErrNeedSingleArgument
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
	role, ok := raw.(*model.Role)
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
