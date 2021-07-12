package model

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	GroupType                        = "group" // also, memdb schema name
	UserInTenantGroupIndex           = "user_in_tenant_group_index"
	ServiceAccountInTenantGroupIndex = "service_account_in_tenant_group_index"
	GroupInTenantGroupIndex          = "group_in_tenant_group_index"
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
					UserInTenantGroupIndex: {
						Name:         UserInTenantGroupIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &subjectInTenantGroupIndexer{
							subjectFieldName: "Users",
						},
					},
					ServiceAccountInTenantGroupIndex: {
						Name:         ServiceAccountInTenantGroupIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &subjectInTenantGroupIndexer{
							subjectFieldName: "ServiceAccounts",
						},
					},
					GroupInTenantGroupIndex: {
						Name:         GroupInTenantGroupIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &subjectInTenantGroupIndexer{
							subjectFieldName: "Groups",
						},
					},
				},
			},
		},
	}
}

type Group struct {
	UUID           GroupUUID  `json:"uuid"` // PK
	TenantUUID     TenantUUID `json:"tenant_uuid"`
	Version        string     `json:"resource_version"`
	Identifier     string     `json:"identifier"`
	FullIdentifier string     `json:"full_identifier"`

	Users           []UserUUID           `json:"-"`
	Groups          []GroupUUID          `json:"-"`
	ServiceAccounts []ServiceAccountUUID `json:"-"`
	Subjects        []SubjectNotation    `json:"subjects"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"extensions"`
}

func (u *Group) ObjType() string {
	return GroupType
}

func (u *Group) ObjId() string {
	return u.UUID
}

// generic: <identifier>@group.<tenant_identifier>
// builtin: <identifier>@<builtin_group_type>.group.<tenant_identifier>
func CalcGroupFullIdentifier(g *Group, tenant *Tenant) string {
	name := g.Identifier
	domain := "group." + tenant.Identifier
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

func (r *GroupRepository) save(group *Group) error {
	return r.db.Insert(GroupType, group)
}

func (r *GroupRepository) Create(group *Group) error {
	tenant, err := r.tenantRepo.GetByID(group.TenantUUID)
	if err != nil {
		return err
	}

	if group.Version != "" {
		return ErrBadVersion
	}
	if group.Origin == "" {
		return ErrBadOrigin
	}
	group.Version = NewResourceVersion()
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	if err := r.fillSubjects(group); err != nil {
		return err
	}

	return r.save(group)
}

func (r *GroupRepository) GetByID(id GroupUUID) (*Group, error) {
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
	stored, err := r.GetByID(group.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != group.TenantUUID {
		return ErrNotFound
	}
	if stored.Origin != group.Origin {
		return ErrBadOrigin
	}
	if stored.Version != group.Version {
		return ErrBadVersion
	}
	group.Version = NewResourceVersion()

	// Update

	tenant, err := r.tenantRepo.GetByID(group.TenantUUID)
	if err != nil {
		return err
	}
	group.FullIdentifier = CalcGroupFullIdentifier(group, tenant)

	if err := r.fillSubjects(group); err != nil {
		return err
	}

	return r.save(group)
}

func (r *GroupRepository) fillSubjects(g *Group) error {
	subj, err := NewSubjectsFetcher(r.db, g.Subjects).Fetch()
	if err != nil {
		return err
	}
	g.Groups = subj.Groups
	g.ServiceAccounts = subj.ServiceAccounts
	g.Users = subj.Users
	return nil
}

/*
TODO Clean from everywhere:
	* other groups
	* role_bindings
	* approvals
	* identity_sharings
*/
func (r *GroupRepository) Delete(origin ObjectOrigin, id GroupUUID) error {
	group, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if group.Origin != origin {
		return ErrBadOrigin
	}
	return r.db.Delete(GroupType, group)
}

func (r *GroupRepository) delete(id string) error {
	group, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(GroupType, group)
}

func (r *GroupRepository) List(tenantID TenantUUID) ([]*Group, error) {
	iter, err := r.db.Get(GroupType, TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	list := []*Group{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		g := raw.(*Group)
		list = append(list, g)
	}
	return list, nil
}

func (r *GroupRepository) DeleteByTenant(tenantUUID TenantUUID) error {
	_, err := r.db.DeleteAll(GroupType, TenantForeignPK, tenantUUID)
	return err
}

func (r *GroupRepository) SetExtension(ext *Extension) error {
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

func (r *GroupRepository) UnsetExtension(origin ObjectOrigin, uuid GroupUUID) error {
	obj, err := r.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return r.save(obj)
}

func (r *GroupRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.delete(objID)
	}

	gr := &Group{}
	err := json.Unmarshal(data, gr)
	if err != nil {
		return err
	}

	return r.save(gr)
}

func extractGroupUUIDs(iter memdb.ResultIterator) map[GroupUUID]struct{} {
	ids := map[GroupUUID]struct{}{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		g := raw.(*Group)
		ids[g.UUID] = struct{}{}
	}
	return ids
}

type subjectInTenantGroupIndexer struct {
	subjectFieldName string
}

func (_ subjectInTenantGroupIndexer) FromArgs(args ...interface{}) ([]byte, error) {
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

func (s subjectInTenantGroupIndexer) FromObject(raw interface{}) (bool, [][]byte, error) {
	usersLabel := "Users"
	serviceAccountsLabel := "ServiceAccounts"
	groupsLabel := "Groups"
	validSubjectFieldNames := map[string]struct{}{usersLabel: {}, serviceAccountsLabel: {}, groupsLabel: {}}
	if _, valid := validSubjectFieldNames[s.subjectFieldName]; !valid {
		return false, nil, fmt.Errorf("invalid subject_field_name: %s", s.subjectFieldName)
	}
	group, ok := raw.(*Group)
	if !ok {
		return false, nil, fmt.Errorf("format error: need Role type, actual passed %#v", raw)
	}
	result := [][]byte{}
	tenantUUID := group.TenantUUID
	switch s.subjectFieldName {
	case usersLabel:
		for i := range group.Users {
			result = append(result, []byte(tenantUUID+group.Users[i]+"\x00"))
		}
	case serviceAccountsLabel:
		for i := range group.ServiceAccounts {
			result = append(result, []byte(tenantUUID+group.ServiceAccounts[i]+"\x00"))
		}
	case groupsLabel:
		for i := range group.Groups {
			result = append(result, []byte(tenantUUID+group.Groups[i]+"\x00"))
		}
	}
	if len(result) == 0 {
		return false, nil, nil
	}
	return true, result, nil
}

func (r *GroupRepository) findDirectParentGroupsByUserUUID(tenantUUID TenantUUID, userUUID UserUUID) (map[GroupUUID]struct{}, error) {
	iter, err := r.db.Get(GroupType, UserInTenantGroupIndex, tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}

	return extractGroupUUIDs(iter), nil
}

func (r *GroupRepository) findDirectParentGroupsByServiceAccountUUID(tenantUUID TenantUUID, serviceAccountUUID ServiceAccountUUID) (map[GroupUUID]struct{}, error) {
	iter, err := r.db.Get(GroupType, ServiceAccountInTenantGroupIndex, tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return extractGroupUUIDs(iter), nil
}

func (r *GroupRepository) findDirectParentGroupsByGroupUUID(tenantUUID TenantUUID, groupUUID GroupUUID) (map[GroupUUID]struct{}, error) {
	iter, err := r.db.Get(GroupType, GroupInTenantGroupIndex, tenantUUID, groupUUID)
	if err != nil {
		return nil, err
	}
	return extractGroupUUIDs(iter), nil
}

// returns map with found parent uuids and originally passed uuids
func (r *GroupRepository) findAllParentGroupsForGroupUUIDs(tenantUUID TenantUUID, groupUUIDs map[GroupUUID]struct{}) (map[GroupUUID]struct{}, error) {
	resultGroupsSet := groupUUIDs
	currentGroupsSet := groupUUIDs
	for len(currentGroupsSet) != 0 {
		nextSet := map[GroupUUID]struct{}{}
		for currentGroupUUID := range currentGroupsSet {
			candidates, err := r.findDirectParentGroupsByGroupUUID(tenantUUID, currentGroupUUID)
			if err != nil {
				return nil, err
			}
			for candidate := range candidates {
				if _, found := resultGroupsSet[candidate]; !found {
					resultGroupsSet[candidate] = struct{}{}
					nextSet[candidate] = struct{}{}
				}
			}
		}
		currentGroupsSet = nextSet
	}
	return resultGroupsSet, nil
}

func (r *GroupRepository) FindAllParentGroupsForUserUUID(tenantUUID TenantUUID, userUUID UserUUID) (map[GroupUUID]struct{}, error) {
	groups, err := r.findDirectParentGroupsByUserUUID(tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}
	return r.findAllParentGroupsForGroupUUIDs(tenantUUID, groups)
}

func (r *GroupRepository) FindAllParentGroupsForServiceAccountUUID(tenantUUID TenantUUID, serviceAccountUUID ServiceAccountUUID) (map[GroupUUID]struct{}, error) {
	groups, err := r.findDirectParentGroupsByServiceAccountUUID(tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return r.findAllParentGroupsForGroupUUIDs(tenantUUID, groups)
}
