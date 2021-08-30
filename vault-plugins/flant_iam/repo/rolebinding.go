package repo

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	RoleBindingForeignPK                   = "role_binding_uuid"
	UserInTenantRoleBindingIndex           = "user_in_tenant_role_binding"
	ServiceAccountInTenantRoleBindingIndex = "service_account_in_tenant_role_binding"
	GroupInTenantRoleBindingIndex          = "group_in_tenant_role_binding"
	ProjectInTenantRoleBindingIndex        = "project_in_tenant_role_binding"
	RoleInTenantRoleBindingIndex           = "role_in_tenant_role_binding"
	TenantUUIDRoleBindingIdIndex           = "tenant_uuid_role_binding_id"
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
					UserInTenantRoleBindingIndex: {
						Name:         UserInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memberInTenantRoleBindingIndexer{
							memberFieldName: "Users",
						},
					},
					ServiceAccountInTenantRoleBindingIndex: {
						Name:         ServiceAccountInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memberInTenantRoleBindingIndexer{
							memberFieldName: "ServiceAccounts",
						},
					},
					GroupInTenantRoleBindingIndex: {
						Name:         GroupInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memberInTenantRoleBindingIndexer{
							memberFieldName: "Groups",
						},
					},
					ProjectInTenantRoleBindingIndex: {
						Name:         ProjectInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memberInTenantRoleBindingIndexer{
							memberFieldName: "Projects",
						},
					},
					RoleInTenantRoleBindingIndex: {
						Name:         RoleInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &roleInTenantRoleBindingIndexer{},
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

// memberInTenantRoleBindingIndexer build index tenantUUID+rb.ServiceAccounts[i].UUID, several indexes for one record
type memberInTenantRoleBindingIndexer struct {
	memberFieldName string
}

func (_ memberInTenantRoleBindingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, model.ErrNeedDoubleArgument
	}
	tenantUUID, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be a string: %#v", args[0])
	}
	memberUUID, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be a string: %#v", args[1])
	}
	// Add the null character as a terminator
	return []byte(tenantUUID + memberUUID + "\x00"), nil
}

func (s memberInTenantRoleBindingIndexer) FromObject(raw interface{}) (bool, [][]byte, error) {
	usersLabel := "Users"
	serviceAccountsLabel := "ServiceAccounts"
	groupsLabel := "Groups"
	projectLabel := "Projects"
	validMemberFieldNames := map[string]struct{}{
		usersLabel: {}, serviceAccountsLabel: {},
		groupsLabel: {}, projectLabel: {},
	}
	if _, valid := validMemberFieldNames[s.memberFieldName]; !valid {
		return false, nil, fmt.Errorf("invalid member_field_name: %s", s.memberFieldName)
	}
	rb, ok := raw.(*model.RoleBinding)
	if !ok {
		return false, nil, fmt.Errorf("format error: need RoleBinding type, actual passed %#v", raw)
	}
	result := [][]byte{}
	tenantUUID := rb.TenantUUID
	switch s.memberFieldName {
	case usersLabel:
		for i := range rb.Users {
			result = append(result, []byte(tenantUUID+rb.Users[i]+"\x00"))
		}
	case serviceAccountsLabel:
		for i := range rb.ServiceAccounts {
			result = append(result, []byte(tenantUUID+rb.ServiceAccounts[i]+"\x00"))
		}
	case groupsLabel:
		for i := range rb.Groups {
			result = append(result, []byte(tenantUUID+rb.Groups[i]+"\x00"))
		}
	case projectLabel:
		for i := range rb.Projects {
			result = append(result, []byte(tenantUUID+rb.Projects[i]+"\x00"))
		}
	}
	if len(result) == 0 {
		return false, nil, nil
	}
	return true, result, nil
}

func extractRoleBindings(iter memdb.ResultIterator) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
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
		rbs[rb.UUID] = rb
	}
	return rbs, nil
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantUser(tenantUUID model.TenantUUID,
	userUUID model.UserUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, UserInTenantRoleBindingIndex, tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantServiceAccount(tenantUUID model.TenantUUID,
	serviceAccountUUID model.ServiceAccountUUID) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, ServiceAccountInTenantRoleBindingIndex, tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) findDirectRoleBindingsForTenantGroup(tenantUUID model.TenantUUID, groupUUID model.GroupUUID) (
	map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, GroupInTenantRoleBindingIndex, tenantUUID, groupUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantGroups(tenantUUID model.TenantUUID, groupUUIDs ...model.GroupUUID) (
	map[model.RoleBindingUUID]*model.RoleBinding, error) {
	rbs := map[model.RoleBindingUUID]*model.RoleBinding{}
	for _, groupUUID := range groupUUIDs {
		partRBs, err := r.findDirectRoleBindingsForTenantGroup(tenantUUID, groupUUID)
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

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantProject(tenantUUID model.TenantUUID, projectUUID model.ProjectUUID) (
	map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, ProjectInTenantRoleBindingIndex, tenantUUID, projectUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

// roleInTenantRoleBindingIndexer build index tenantUUID+rb.Roles[i].Name, several indexes for one record
type roleInTenantRoleBindingIndexer struct{}

func (roleInTenantRoleBindingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, model.ErrNeedDoubleArgument
	}
	tenantUUID, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be a string: %#v", args[0])
	}
	roleName, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be a string: %#v", args[1])
	}
	// Add the null character as a terminator
	return []byte(tenantUUID + roleName + "\x00"), nil
}

func (roleInTenantRoleBindingIndexer) FromObject(raw interface{}) (bool, [][]byte, error) {
	rb, ok := raw.(*model.RoleBinding)
	if !ok {
		return false, nil, fmt.Errorf("format error: need RoleBinding type, actual passed %#v", raw)
	}
	result := [][]byte{}
	tenantUUID := rb.TenantUUID
	for i := range rb.Roles {
		result = append(result, []byte(tenantUUID+rb.Roles[i].Name+"\x00"))
	}
	if len(result) == 0 {
		return false, nil, nil
	}
	return true, result, nil
}

func (r *RoleBindingRepository) findDirectRoleBindingsForRole(tenantUUID model.TenantUUID, role model.RoleName) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, RoleInTenantRoleBindingIndex, tenantUUID, role)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForRoles(tenantUUID model.TenantUUID, roles ...model.RoleName) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	roleBindings := map[model.RoleBindingUUID]*model.RoleBinding{}
	for _, role := range roles {
		partRoleBindings, err := r.findDirectRoleBindingsForRole(tenantUUID, role)
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
