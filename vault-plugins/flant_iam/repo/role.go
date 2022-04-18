package repo

import (
	"encoding/json"
	"fmt"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const (
	IncludedRolesIndex     = "included_roles_index"
	RoleScopeIndex         = "scope"
	FeatureFlagInRoleIndex = "feature_flag_in_role_index"
)

func RoleSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.RoleType: {
				Name: model.RoleType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Name",
						},
					},
					RoleScopeIndex: {
						Name: RoleScopeIndex,
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Scope",
						},
					},
					IncludedRolesIndex: {
						Name:         IncludedRolesIndex,
						AllowMissing: true,
						Unique:       false,
						Indexer: &memdb.CustomTypeSliceFieldIndexer{
							Field: "IncludedRoles",
							FromCustomType: func(customTypeValue interface{}) ([]byte, error) {
								obj, ok := customTypeValue.(model.IncludedRole)
								if !ok {
									return nil, fmt.Errorf("need IncludedRole, actual:%T", customTypeValue)
								}
								return []byte(obj.Name), nil
							},
						},
					},
					FeatureFlagInRoleIndex: {
						Name:         FeatureFlagInRoleIndex,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "RequireOneOfFeatureFlags",
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.RoleType: {
				{
					OriginalDataTypeFieldName:     "RequireOneOfFeatureFlags",
					RelatedDataType:               model.FeatureFlagType,
					RelatedDataTypeFieldIndexName: PK,
				},
				{
					OriginalDataTypeFieldName:     "IncludedRoles",
					RelatedDataType:               model.RoleType,
					RelatedDataTypeFieldIndexName: PK,
					BuildRelatedCustomType: func(originalFieldValue interface{}) (customTypeValue interface{}, err error) {
						obj, ok := originalFieldValue.(model.IncludedRole)
						if !ok {
							return nil, fmt.Errorf("need IncludedRole, actual:%T", originalFieldValue)
						}
						return obj.Name, nil
					},
				},
			},
		},
		CheckingRelations: map[string][]memdb.Relation{
			model.RoleType: {
				{
					OriginalDataTypeFieldName:     "Name",
					RelatedDataType:               model.RoleType,
					RelatedDataTypeFieldIndexName: IncludedRolesIndex,
					BuildRelatedCustomType: func(originalFieldValue interface{}) (customTypeValue interface{}, err error) {
						name, ok := originalFieldValue.(string)
						if !ok {
							return nil, fmt.Errorf("need string type, got: %T", originalFieldValue)
						}
						return model.IncludedRole{
							Name: name,
						}, nil
					},
				},
				{
					OriginalDataTypeFieldName:     "Name",
					RelatedDataType:               model.RoleBindingType,
					RelatedDataTypeFieldIndexName: RoleInRoleBindingIndex,
					BuildRelatedCustomType: func(originalFieldValue interface{}) (customTypeValue interface{}, err error) {
						name, ok := originalFieldValue.(string)
						if !ok {
							return nil, fmt.Errorf("need string type, got: %T", originalFieldValue)
						}
						return model.BoundRole{
							Name: name,
						}, nil
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
		return nil, consts.ErrNotFound
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
		if showArchived || obj.NotArchived() {
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

func (r *RoleRepository) Delete(roleID model.RoleName, archiveMark memdb.ArchiveMark) error {
	role, err := r.GetByID(roleID)
	if err != nil {
		return err
	}
	if role.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(model.RoleType, role, archiveMark)
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

// RoleChain represents chain of some role nesting for options resolving
type RoleChain struct {
	// Role name at the start of the chain
	RoleName model.RoleName
	// OptionsTemplates which should be applied, from [0] to the last
	OptionsTemplates []string
}

func (r *RoleRepository) FindDirectParentRoles(roleID model.RoleName) (map[model.RoleName]RoleChain, error) {
	iter, err := r.db.Get(model.RoleType, IncludedRolesIndex, model.IncludedRole{Name: roleID})
	if err != nil {
		return nil, err
	}
	result := map[model.RoleName]RoleChain{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		role := raw.(*model.Role)
		if role.NotArchived() {
			for _, includedRole := range role.IncludedRoles {
				if includedRole.Name == roleID {
					result[role.Name] = RoleChain{
						RoleName:         role.Name,
						OptionsTemplates: []string{includedRole.OptionsTemplate},
					}
					break
				}
			}
		}
	}
	return result, nil
}

func (r *RoleRepository) FindAllAncestorsRoles(roleID model.RoleName) (map[model.RoleName]RoleChain, error) {
	result := map[model.RoleName]RoleChain{roleID: {RoleName: roleID}}
	role, err := r.GetByID(roleID)
	if err != nil {
		return nil, err
	}
	if role.Archived() {
		return result, nil
	}
	currentSetOfChains := map[model.RoleName]RoleChain{roleID: {RoleName: roleID}}
	for len(currentSetOfChains) != 0 {
		nextSet := map[model.RoleName]RoleChain{}
		for currentRole, currentOptionsChain := range currentSetOfChains {
			candidates, err := r.FindDirectParentRoles(currentRole)
			if err != nil {
				return nil, err
			}
			for candidate, candidateChain := range candidates {
				if _, found := result[candidate]; !found && candidate != roleID {
					chain := RoleChain{
						RoleName:         candidate,
						OptionsTemplates: append(candidateChain.OptionsTemplates, currentOptionsChain.OptionsTemplates...),
					}
					result[candidate] = chain
					nextSet[candidate] = chain
				}
			}
		}
		currentSetOfChains = nextSet
	}
	return result, nil
}
