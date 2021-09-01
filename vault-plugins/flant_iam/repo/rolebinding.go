package repo

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	RoleBindingForeignPK             = "role_binding_uuid"
	UserInRoleBindingIndex           = "user_in_role_binding"
	ServiceAccountInRoleBindingIndex = "service_account_in_role_binding"
	GroupInRoleBindingIndex          = "group_in_role_binding"
	ProjectInRoleBindingIndex        = "project_in_role_binding"
	RoleInRoleBindingIndex           = "role_in_role_binding"
	TenantUUIDRoleBindingIdIndex     = "tenant_uuid_role_binding_id"
)

func RoleBindingSchema() *memdb.DBSchema {
	var tenantUUIDRoleBindingIdIndexer []memdb.Indexer

	tenantUUIDIndexer := &memdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	tenantUUIDRoleBindingIdIndexer = append(tenantUUIDRoleBindingIdIndexer, tenantUUIDIndexer)

	groupIdIndexer := &memdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	tenantUUIDRoleBindingIdIndexer = append(tenantUUIDRoleBindingIdIndexer, groupIdIndexer)

	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			model.RoleBindingType: {
				Name: model.RoleBindingType,
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
					UserInRoleBindingIndex: {
						Name:         UserInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memdb.StringSliceFieldIndex{
							Field: "Users",
						},
					},
					ServiceAccountInRoleBindingIndex: {
						Name:         ServiceAccountInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memdb.StringSliceFieldIndex{
							Field: "ServiceAccounts",
						},
					},
					GroupInRoleBindingIndex: {
						Name:         GroupInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memdb.StringSliceFieldIndex{
							Field: "Groups",
						},
					},
					ProjectInRoleBindingIndex: {
						Name:         ProjectInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memdb.StringSliceFieldIndex{
							Field: "Projects",
						},
					},
					RoleInRoleBindingIndex: {
						Name:         RoleInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &roleInRoleBindingIndexer{},
					},
					TenantUUIDRoleBindingIdIndex: {
						Name:    TenantUUIDRoleBindingIdIndex,
						Indexer: &memdb.CompoundIndex{Indexes: tenantUUIDRoleBindingIdIndexer},
					},
				},
			},
		},
	}
}

type RoleBindingRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleBindingRepository(tx *io.MemoryStoreTxn) *RoleBindingRepository {
	return &RoleBindingRepository{db: tx}
}

func (r *RoleBindingRepository) save(rb *model.RoleBinding) error {
	return r.db.Insert(model.RoleBindingType, rb)
}

func (r *RoleBindingRepository) Create(rb *model.RoleBinding) error {
	return r.save(rb)
}

func (r *RoleBindingRepository) GetRawByID(id model.RoleBindingUUID) (interface{}, error) {
	raw, err := r.db.First(model.RoleBindingType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *RoleBindingRepository) GetByID(id model.RoleBindingUUID) (*model.RoleBinding, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.RoleBinding), err
}

func (r *RoleBindingRepository) Update(rb *model.RoleBinding) error {
	_, err := r.GetByID(rb.UUID)
	if err != nil {
		return err
	}
	return r.save(rb)
}

func (r *RoleBindingRepository) Delete(id model.RoleBindingUUID, archivingTimestamp model.UnixTime, archivingHash int64) error {
	rb, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if rb.IsDeleted() {
		return model.ErrIsArchived
	}
	rb.ArchivingTimestamp = archivingTimestamp
	rb.ArchivingHash = archivingHash
	return r.Update(rb)
}

func (r *RoleBindingRepository) List(tenantUUID model.TenantUUID, showArchived bool) ([]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.RoleBinding{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.RoleBinding)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *RoleBindingRepository) ListIDs(tenantID model.TenantUUID, showArchived bool) ([]model.RoleBindingUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.RoleBindingUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *RoleBindingRepository) Iter(action func(*model.RoleBinding) (bool, error)) error {
	iter, err := r.db.Get(model.RoleBindingType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.RoleBinding)
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

func (r *RoleBindingRepository) Sync(_ string, data []byte) error {
	rb := &model.RoleBinding{}
	err := json.Unmarshal(data, rb)
	if err != nil {
		return err
	}

	return r.save(rb)
}

func (r *RoleBindingRepository) GetByIdentifier(tenantUUID, identifier string) (*model.RoleBinding, error) {
	raw, err := r.db.First(model.RoleBindingType, TenantUUIDRoleBindingIdIndex, tenantUUID, identifier)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	roleBinding := raw.(*model.RoleBinding)
	return roleBinding, nil
}

func extractRoleBindings(iter memdb.ResultIterator, showArchived bool) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	rbs := map[model.RoleBindingUUID]*model.RoleBinding{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		rb, ok := raw.(*model.RoleBinding)
		if !ok {
			return nil, fmt.Errorf("need type RoleBindig, actually passed: %#v", raw)
		}
		if !showArchived && rb.ArchivingHash != 0 {
			continue
		}
		rbs[rb.UUID] = rb
	}
	return rbs, nil
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForUser(
	userUUID model.UserUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, UserInRoleBindingIndex, userUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter, false)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForServiceAccount(
	serviceAccountUUID model.ServiceAccountUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, ServiceAccountInRoleBindingIndex, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter, false)
}

func (r *RoleBindingRepository) findDirectRoleBindingsForGroup(groupUUID model.GroupUUID) (
	map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, GroupInRoleBindingIndex, groupUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter, false)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForGroups(groupUUIDs ...model.GroupUUID) (
	map[model.RoleBindingUUID]*model.RoleBinding, error) {
	rbs := map[model.RoleBindingUUID]*model.RoleBinding{}
	for _, groupUUID := range groupUUIDs {
		partRBs, err := r.findDirectRoleBindingsForGroup(groupUUID)
		if err != nil {
			return nil, err
		}
		for uuid, rb := range partRBs {
			if _, found := rbs[uuid]; !found {
				rbs[uuid] = rb
			}
		}
	}
	return rbs, nil
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForProject(projectUUID model.ProjectUUID) (
	map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, ProjectInRoleBindingIndex, projectUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter, false)
}

// roleInRoleBindingIndexer builds index rb.Roles[i].Name, several indexes for one record
type roleInRoleBindingIndexer struct{}

func (roleInRoleBindingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, model.ErrNeedSingleArgument
	}
	roleName, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be a string: %#v", args[1])
	}
	// Add the null character as a terminator
	return []byte(roleName + "\x00"), nil
}

func (roleInRoleBindingIndexer) FromObject(raw interface{}) (bool, [][]byte, error) {
	rb, ok := raw.(*model.RoleBinding)
	if !ok {
		return false, nil, fmt.Errorf("format error: need RoleBinding type, actual passed %#v", raw)
	}
	result := [][]byte{}
	for i := range rb.Roles {
		result = append(result, []byte(rb.Roles[i].Name+"\x00"))
	}
	if len(result) == 0 {
		return false, nil, nil
	}
	return true, result, nil
}

func (r *RoleBindingRepository) findDirectRoleBindingsForRole(role model.RoleName) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, RoleInRoleBindingIndex, role)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter, false)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForRoles(roles ...model.RoleName) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	roleBindings := map[model.RoleBindingUUID]*model.RoleBinding{}
	for _, role := range roles {
		partRoleBindings, err := r.findDirectRoleBindingsForRole(role)
		if err != nil {
			return nil, err
		}
		for uuid, rb := range partRoleBindings {
			if _, found := roleBindings[uuid]; !found {
				roleBindings[uuid] = rb
			}
		}
	}
	return roleBindings, nil
}
