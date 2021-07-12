package model

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	RoleBindingType                        = "role_binding" // also, memdb schema name
	UserInTenantRoleBindingIndex           = "user_in_tenant_role_binding"
	ServiceAccountInTenantRoleBindingIndex = "service_account_in_tenant_role_binding"
	GroupInTenantRoleBindingIndex          = "group_in_tenant_role_binding"
	ProjectInTenantRoleBindingIndex        = "project_in_tenant_role_binding"
	RoleInTenantRoleBindingIndex           = "role_in_tenant_role_binding"
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
					UserInTenantRoleBindingIndex: {
						Name:         UserInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &subjectInTenantRoleBindingIndexer{
							subjectFieldName: "Users",
						},
					},
					ServiceAccountInTenantRoleBindingIndex: {
						Name:         ServiceAccountInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &subjectInTenantRoleBindingIndexer{
							subjectFieldName: "ServiceAccounts",
						},
					},
					GroupInTenantRoleBindingIndex: {
						Name:         GroupInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &subjectInTenantRoleBindingIndexer{
							subjectFieldName: "Groups",
						},
					},
					ProjectInTenantRoleBindingIndex: {
						Name:         ProjectInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &subjectInTenantRoleBindingIndexer{
							subjectFieldName: "Projects",
						},
					},
					RoleInTenantRoleBindingIndex: {
						Name:         RoleInTenantRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &roleInTenantRoleBindingIndexer{},
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
	Scope   RoleScope              `json:"scope"`
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

type subjectInTenantRoleBindingIndexer struct {
	subjectFieldName string
}

func (_ subjectInTenantRoleBindingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, ErrNeedDoubleArgument
	}
	tenantUUID, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be a string: %#v", args[0])
	}
	subjectUUID, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("argument must be a string: %#v", args[1])
	}
	// Add the null character as a terminator
	return []byte(tenantUUID + subjectUUID + "\x00"), nil
}

func (s subjectInTenantRoleBindingIndexer) FromObject(raw interface{}) (bool, [][]byte, error) {
	usersLabel := "Users"
	serviceAccountsLabel := "ServiceAccounts"
	groupsLabel := "Groups"
	projectLabel := "Projects"
	validSubjectFieldNames := map[string]struct{}{usersLabel: {}, serviceAccountsLabel: {}, groupsLabel: {}}
	if _, valid := validSubjectFieldNames[s.subjectFieldName]; !valid {
		return false, nil, fmt.Errorf("invalid subject_field_name: %s", s.subjectFieldName)
	}
	rb, ok := raw.(*RoleBinding)
	if !ok {
		return false, nil, fmt.Errorf("format error: need RoleBinding type, actual passed %#v", raw)
	}
	result := [][]byte{}
	tenantUUID := rb.TenantUUID
	switch s.subjectFieldName {
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

func extractRoleBindings(iter memdb.ResultIterator) (map[RoleBindingUUID]*RoleBinding, error) {
	rbs := map[RoleBindingUUID]*RoleBinding{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		rb, ok := raw.(*RoleBinding)
		if !ok {
			return nil, fmt.Errorf("need type RoleBindig, actually passed: %#v", raw)
		}
		rbs[rb.UUID] = rb
	}
	return rbs, nil
}

func extractRoleBindingUUIDs(iter memdb.ResultIterator) (map[RoleBindingUUID]struct{}, error) {
	uuids := map[RoleBindingUUID]struct{}{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		rb, ok := raw.(*RoleBinding)
		if !ok {
			return nil, fmt.Errorf("need type RoleBindig, actually passed: %#v", raw)
		}
		uuids[rb.UUID] = struct{}{}
	}
	return uuids, nil
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantUser(tenantUUID TenantUUID, userUUID UserUUID) (map[RoleBindingUUID]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, UserInTenantRoleBindingIndex, tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantServiceAccount(tenantUUID TenantUUID, serviceAccountUUID ServiceAccountUUID) (map[RoleBindingUUID]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, ServiceAccountInTenantRoleBindingIndex, tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) findDirectRoleBindingsForTenantGroup(tenantUUID TenantUUID, groupUUID GroupUUID) (map[RoleBindingUUID]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, GroupInTenantRoleBindingIndex, tenantUUID, groupUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantGroups(tenantUUID TenantUUID, groupUUIDs ...GroupUUID) (map[RoleBindingUUID]*RoleBinding, error) {
	rbs := map[RoleBindingUUID]*RoleBinding{}
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

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantProject(tenantUUID TenantUUID, projectUUID ProjectUUID) (map[RoleBindingUUID]struct{}, error) {
	iter, err := r.db.Get(RoleBindingType, ProjectInTenantRoleBindingIndex, tenantUUID, projectUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindingUUIDs(iter)
}

type roleInTenantRoleBindingIndexer struct{}

func (roleInTenantRoleBindingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, ErrNeedDoubleArgument
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
	rb, ok := raw.(*RoleBinding)
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

func (r *RoleBindingRepository) findDirectRoleBindingsForRole(tenantUUID TenantUUID, role RoleName) (map[RoleBindingUUID]struct{}, error) {
	iter, err := r.db.Get(RoleBindingType, RoleInTenantRoleBindingIndex, tenantUUID, role)
	if err != nil {
		return nil, err
	}
	return extractRoleBindingUUIDs(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForRoles(tenantUUID TenantUUID, roles ...RoleName) (map[RoleBindingUUID]struct{}, error) {
	uuids := map[RoleBindingUUID]struct{}{}
	for _, role := range roles {
		partUUIDs, err := r.findDirectRoleBindingsForRole(tenantUUID, role)
		if err != nil {
			return nil, err
		}
		for uuid := range partUUIDs {
			if _, found := uuids[uuid]; !found {
				uuids[uuid] = struct{}{}
			}
		}
	}
	return uuids, nil
}
