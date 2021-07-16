package model

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
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

type RoleBindingObjectType string

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

//go:generate go run gen_repository.go -type RoleBinding -parentType Tenant
type RoleBinding struct {
	UUID       RoleBindingUUID `json:"uuid"` // PK
	TenantUUID TenantUUID      `json:"tenant_uuid"`
	Version    string          `json:"resource_version"`

	Identifier string `json:"identifier"`

	ValidTill  int64 `json:"valid_till"`
	RequireMFA bool  `json:"require_mfa"`

	Users           []UserUUID           `json:"-"`
	Groups          []GroupUUID          `json:"-"`
	ServiceAccounts []ServiceAccountUUID `json:"-"`
	Members         []MemberNotation     `json:"members"`

	AnyProject bool          `json:"any_project"`
	Projects   []ProjectUUID `json:"projects"`

	Roles []BoundRole `json:"roles"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"-"`
}

type BoundRole struct {
	Name    RoleName               `json:"name"`
	Options map[string]interface{} `json:"options"`
}

func (r *RoleBindingRepository) GetByIdentifier(tenantUUID, identifier string) (*RoleBinding, error) {
	raw, err := r.db.First(RoleBindingType, TenantUUIDRoleBindingIdIndex, tenantUUID, identifier)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	roleBinding := raw.(*RoleBinding)
	return roleBinding, nil
}

type memberInTenantRoleBindingIndexer struct {
	memberFieldName string
}

func (_ memberInTenantRoleBindingIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, ErrNeedDoubleArgument
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
	rb, ok := raw.(*RoleBinding)
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

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantUser(tenantUUID TenantUUID,
	userUUID UserUUID) (map[RoleBindingUUID]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, UserInTenantRoleBindingIndex, tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantServiceAccount(tenantUUID TenantUUID,
	serviceAccountUUID ServiceAccountUUID) (map[RoleBindingUUID]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, ServiceAccountInTenantRoleBindingIndex, tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) findDirectRoleBindingsForTenantGroup(tenantUUID TenantUUID, groupUUID GroupUUID) (
	map[RoleBindingUUID]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, GroupInTenantRoleBindingIndex, tenantUUID, groupUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantGroups(tenantUUID TenantUUID, groupUUIDs ...GroupUUID) (
	map[RoleBindingUUID]*RoleBinding, error) {
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

func (r *RoleBindingRepository) FindDirectRoleBindingsForTenantProject(tenantUUID TenantUUID, projectUUID ProjectUUID) (
	map[RoleBindingUUID]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, ProjectInTenantRoleBindingIndex, tenantUUID, projectUUID)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
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

func (r *RoleBindingRepository) findDirectRoleBindingsForRole(tenantUUID TenantUUID, role RoleName) (map[RoleBindingUUID]*RoleBinding, error) {
	iter, err := r.db.Get(RoleBindingType, RoleInTenantRoleBindingIndex, tenantUUID, role)
	if err != nil {
		return nil, err
	}
	return extractRoleBindings(iter)
}

func (r *RoleBindingRepository) FindDirectRoleBindingsForRoles(tenantUUID TenantUUID, roles ...RoleName) (map[RoleBindingUUID]*RoleBinding, error) {
	roleBindings := map[RoleBindingUUID]*RoleBinding{}
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
