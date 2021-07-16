package model

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	UserInTenantGroupIndex           = "user_in_tenant_group_index"
	ServiceAccountInTenantGroupIndex = "service_account_in_tenant_group_index"
	GroupInTenantGroupIndex          = "group_in_tenant_group_index"
	TenantUUIDGroupIdIndex           = "tenant_uuid_group_id"
)

type GroupObjectType string

func GroupSchema() *memdb.DBSchema {
	var tenantUUIDGroupIdIndexer []memdb.Indexer

	tenantUUIDIndexer := &memdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	tenantUUIDGroupIdIndexer = append(tenantUUIDGroupIdIndexer, tenantUUIDIndexer)

	groupIdIndexer := &memdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	tenantUUIDGroupIdIndexer = append(tenantUUIDGroupIdIndexer, groupIdIndexer)

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
						Indexer: &memberInTenantGroupIndexer{
							memberFieldName: "Users",
						},
					},
					ServiceAccountInTenantGroupIndex: {
						Name:         ServiceAccountInTenantGroupIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memberInTenantGroupIndexer{
							memberFieldName: "ServiceAccounts",
						},
					},
					GroupInTenantGroupIndex: {
						Name:         GroupInTenantGroupIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &memberInTenantGroupIndexer{
							memberFieldName: "Groups",
						},
					},
					TenantUUIDGroupIdIndex: {
						Name:    TenantUUIDGroupIdIndex,
						Indexer: &memdb.CompoundIndex{Indexes: tenantUUIDGroupIdIndexer},
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
	Members         []MemberNotation     `json:"members"`

	Origin ObjectOrigin `json:"origin"`

	Extensions map[ObjectOrigin]*Extension `json:"-"`
}

const GroupType = "group" // also, memdb schema name

func (u *Group) ObjType() string {
	return GroupType
}

func (u *Group) ObjId() string {
	return u.UUID
}

type GroupRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewGroupRepository(tx *io.MemoryStoreTxn) *GroupRepository {
	return &GroupRepository{db: tx}
}

func (r *GroupRepository) save(group *Group) error {
	return r.db.Insert(GroupType, group)
}

func (r *GroupRepository) Create(group *Group) error {
	return r.save(group)
}

func (r *GroupRepository) GetRawByID(id GroupUUID) (interface{}, error) {
	raw, err := r.db.First(GroupType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *GroupRepository) GetByID(id GroupUUID) (*Group, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*Group), err
}

func (r *GroupRepository) Update(group *Group) error {
	_, err := r.GetByID(group.UUID)
	if err != nil {
		return err
	}
	return r.save(group)
}

func (r *GroupRepository) Delete(id GroupUUID) error {
	group, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(GroupType, group)
}

func (r *GroupRepository) List(tenantUUID TenantUUID) ([]*Group, error) {
	iter, err := r.db.Get(GroupType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*Group{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Group)
		list = append(list, obj)
	}
	return list, nil
}

func (r *GroupRepository) ListIDs(tenantID TenantUUID) ([]GroupUUID, error) {
	objs, err := r.List(tenantID)
	if err != nil {
		return nil, err
	}
	ids := make([]GroupUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *GroupRepository) Iter(action func(*Group) (bool, error)) error {
	iter, err := r.db.Get(GroupType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Group)
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

func (r *GroupRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	group := &Group{}
	err := json.Unmarshal(data, group)
	if err != nil {
		return err
	}

	return r.save(group)
}

func (r *GroupRepository) GetByIdentifier(tenantUUID, identifier string) (*Group, error) {
	raw, err := r.db.First(GroupType, TenantUUIDGroupIdIndex, tenantUUID, identifier)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*Group), err
}

func (r *GroupRepository) FindDirectParentGroupsByUserUUID(tenantUUID TenantUUID, userUUID UserUUID) (map[GroupUUID]struct{}, error) {
	iter, err := r.db.Get(GroupType, UserInTenantGroupIndex, tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}
	return extractGroupUUIDs(iter)
}

func (r *GroupRepository) FindDirectParentGroupsByServiceAccountUUID(tenantUUID TenantUUID, serviceAccountUUID ServiceAccountUUID) (map[GroupUUID]struct{}, error) {
	iter, err := r.db.Get(GroupType, ServiceAccountInTenantGroupIndex, tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return extractGroupUUIDs(iter)
}

func (r *GroupRepository) FindDirectParentGroupsByGroupUUID(tenantUUID TenantUUID, groupUUID GroupUUID) (map[GroupUUID]struct{}, error) {
	iter, err := r.db.Get(GroupType, GroupInTenantGroupIndex, tenantUUID, groupUUID)
	if err != nil {
		return nil, err
	}
	return extractGroupUUIDs(iter)
}

// returns map with found parent uuids and originally passed uuids
func (r *GroupRepository) FindAllParentGroupsForGroupUUIDs(tenantUUID TenantUUID, groupUUIDs map[GroupUUID]struct{}) (map[GroupUUID]struct{}, error) {
	resultGroupsSet := groupUUIDs
	currentGroupsSet := groupUUIDs
	for len(currentGroupsSet) != 0 {
		nextSet := map[GroupUUID]struct{}{}
		for currentGroupUUID := range currentGroupsSet {
			candidates, err := r.FindDirectParentGroupsByGroupUUID(tenantUUID, currentGroupUUID)
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
	groups, err := r.FindDirectParentGroupsByUserUUID(tenantUUID, userUUID)
	if err != nil {
		return nil, err
	}
	return r.FindAllParentGroupsForGroupUUIDs(tenantUUID, groups)
}

func (r *GroupRepository) FindAllParentGroupsForServiceAccountUUID(tenantUUID TenantUUID, serviceAccountUUID ServiceAccountUUID) (map[GroupUUID]struct{}, error) {
	groups, err := r.FindDirectParentGroupsByServiceAccountUUID(tenantUUID, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return r.FindAllParentGroupsForGroupUUIDs(tenantUUID, groups)
}

func (r *GroupRepository) FindAllParentGroupsForGroupUUID(tenantUUID TenantUUID, groupUUID GroupUUID) (map[GroupUUID]struct{}, error) {
	return r.FindAllParentGroupsForGroupUUIDs(tenantUUID, map[GroupUUID]struct{}{groupUUID: {}})
}

func (r *GroupRepository) FindAllMembersFor(tenantUUID TenantUUID, users []UserUUID, serviceAccounts []ServiceAccountUUID, groups []GroupUUID) (map[UserUUID]struct{}, map[ServiceAccountUUID]struct{}, error) {
	resultUsers := make(map[UserUUID]struct{})
	resultSAs := make(map[ServiceAccountUUID]struct{})

	for _, user := range users {
		resultUsers[user] = struct{}{}
	}
	for _, sa := range serviceAccounts {
		resultSAs[sa] = struct{}{}
	}

	for _, groupUUID := range groups {
		groupsUsers, groupsSAs, err := r.FindAllMembersForGroupUUID(tenantUUID, groupUUID)
		if err != nil {
			return nil, nil, err
		}
		for user := range groupsUsers {
			resultUsers[user] = struct{}{}
		}
		for sa := range groupsSAs {
			resultSAs[sa] = struct{}{}
		}
	}

	return resultUsers, resultSAs, nil
}

func (r *GroupRepository) FindAllMembersForGroupUUID(tenantUUID TenantUUID, groupUUID GroupUUID) (map[UserUUID]struct{}, map[ServiceAccountUUID]struct{}, error) {
	group, err := r.GetByID(groupUUID)
	if err != nil {
		return nil, nil, err
	}
	return r.FindAllMembersForGroup(group)
}

func (r *GroupRepository) FindAllMembersForGroup(group *Group) (map[UserUUID]struct{}, map[ServiceAccountUUID]struct{}, error) {
	return r.FindAllMembersFor(group.TenantUUID, group.Users, group.ServiceAccounts, group.Groups)
}

func extractGroupUUIDs(iter memdb.ResultIterator) (map[GroupUUID]struct{}, error) {
	ids := map[GroupUUID]struct{}{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		g, ok := raw.(*Group)
		if !ok {
			return nil, fmt.Errorf("need type Group, actually passed: %#v", raw)
		}
		ids[g.UUID] = struct{}{}
	}
	return ids, nil
}

type memberInTenantGroupIndexer struct {
	memberFieldName string
}

func (_ memberInTenantGroupIndexer) FromArgs(args ...interface{}) ([]byte, error) {
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

func (s memberInTenantGroupIndexer) FromObject(raw interface{}) (bool, [][]byte, error) {
	usersLabel := "Users"
	serviceAccountsLabel := "ServiceAccounts"
	groupsLabel := "Groups"
	validMemberFieldNames := map[string]struct{}{usersLabel: {}, serviceAccountsLabel: {}, groupsLabel: {}}
	if _, valid := validMemberFieldNames[s.memberFieldName]; !valid {
		return false, nil, fmt.Errorf("invalid member_field_name: %s", s.memberFieldName)
	}
	group, ok := raw.(*Group)
	if !ok {
		return false, nil, fmt.Errorf("format error: need Group type, actual passed %#v", raw)
	}
	result := [][]byte{}
	tenantUUID := group.TenantUUID
	switch s.memberFieldName {
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
