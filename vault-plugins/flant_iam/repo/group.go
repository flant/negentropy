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
	UserInGroupIndex           = "user_in_group_index"
	ServiceAccountInGroupIndex = "service_account_in_group_index"
	GroupInGroupIndex          = "group_in_group_index"
	TenantUUIDGroupIdIndex     = "tenant_uuid_group_id"
)

type GroupObjectType string

func GroupSchema() *memdb.DBSchema {
	var tenantUUIDGroupIdIndexer []hcmemdb.Indexer

	tenantUUIDIndexer := &hcmemdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	tenantUUIDGroupIdIndexer = append(tenantUUIDGroupIdIndexer, tenantUUIDIndexer)

	groupIdIndexer := &hcmemdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	tenantUUIDGroupIdIndexer = append(tenantUUIDGroupIdIndexer, groupIdIndexer)

	return &memdb.DBSchema{

		Tables: map[string]*hcmemdb.TableSchema{
			model.GroupType: {
				Name: model.GroupType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					UserInGroupIndex: {
						Name:         UserInGroupIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "Users",
						},
					},
					ServiceAccountInGroupIndex: {
						Name:         ServiceAccountInGroupIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "ServiceAccounts",
						},
					},
					GroupInGroupIndex: {
						Name:         GroupInGroupIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "Groups",
						},
					},
					TenantUUIDGroupIdIndex: {
						Name:    TenantUUIDGroupIdIndex,
						Indexer: &hcmemdb.CompoundIndex{Indexes: tenantUUIDGroupIdIndexer},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.GroupType: {
				{OriginalDataTypeFieldName: "TenantUUID", RelatedDataType: model.TenantType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "Users", RelatedDataType: model.UserType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "Groups", RelatedDataType: model.GroupType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "ServiceAccounts", RelatedDataType: model.ServiceAccountType, RelatedDataTypeFieldIndexName: PK},
			},
		},
		CascadeDeletes: map[string][]memdb.Relation{
			model.GroupType: {
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.GroupType, RelatedDataTypeFieldIndexName: GroupInGroupIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingType, RelatedDataTypeFieldIndexName: GroupInRoleBindingIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingApprovalType, RelatedDataTypeFieldIndexName: GroupInRoleBindingApprovalIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.IdentitySharingType, RelatedDataTypeFieldIndexName: GroupUUIDIdentitySharingIndex},
			},
		},
	}
}

type GroupRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewGroupRepository(tx *io.MemoryStoreTxn) *GroupRepository {
	return &GroupRepository{db: tx}
}

func (r *GroupRepository) save(group *model.Group) error {
	return r.db.Insert(model.GroupType, group)
}

func (r *GroupRepository) Create(group *model.Group) error {
	return r.save(group)
}

func (r *GroupRepository) GetRawByID(id model.GroupUUID) (interface{}, error) {
	raw, err := r.db.First(model.GroupType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *GroupRepository) GetByID(id model.GroupUUID) (*model.Group, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	g := raw.(*model.Group)
	if g.FixMembers() {
		err = r.save(g)
	}
	return g, err
}

func (r *GroupRepository) Update(group *model.Group) error {
	_, err := r.GetByID(group.UUID)
	if err != nil {
		return err
	}
	return r.save(group)
}

func (r *GroupRepository) Delete(id model.GroupUUID, archiveMark memdb.ArchiveMark) error {
	group, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if group.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(model.GroupType, group, archiveMark)
}

func (r *GroupRepository) CleanChildrenSliceIndexes(id model.GroupUUID) error {
	group, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.CleanChildrenSliceIndexes(model.GroupType, group)
}

func (r *GroupRepository) List(tenantUUID model.TenantUUID, showArchived bool) ([]*model.Group, error) {
	iter, err := r.db.Get(model.GroupType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.Group{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Group)
		if obj.FixMembers() {
			err = r.save(obj)
			if err != nil {
				return nil, err
			}
		}
		if showArchived || obj.NotArchived() {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *GroupRepository) ListIDs(tenantID model.TenantUUID, showArchived bool) ([]model.GroupUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.GroupUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *GroupRepository) Iter(action func(*model.Group) (bool, error)) error {
	iter, err := r.db.Get(model.GroupType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Group)
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
	group := &model.Group{}
	err := json.Unmarshal(data, group)
	if err != nil {
		return err
	}

	return r.save(group)
}

func (r *GroupRepository) GetByIdentifier(tenantUUID, identifier string) (*model.Group, error) {
	raw, err := r.db.First(model.GroupType, TenantUUIDGroupIdIndex, tenantUUID, identifier)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw.(*model.Group), err
}

func (r *GroupRepository) FindDirectParentGroupsByUserUUID(userUUID model.UserUUID) (map[model.GroupUUID]struct{}, error) {
	iter, err := r.db.Get(model.GroupType, UserInGroupIndex, userUUID)
	if err != nil {
		return nil, err
	}
	return extractGroupUUIDs(iter, false)
}

func (r *GroupRepository) FindDirectParentGroupsByServiceAccountUUID(serviceAccountUUID model.ServiceAccountUUID) (map[model.GroupUUID]struct{}, error) {
	iter, err := r.db.Get(model.GroupType, ServiceAccountInGroupIndex, serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return extractGroupUUIDs(iter, false)
}

func (r *GroupRepository) FindDirectParentGroupsByGroupUUID(groupUUID model.GroupUUID) (map[model.GroupUUID]struct{}, error) {
	iter, err := r.db.Get(model.GroupType, GroupInGroupIndex, groupUUID)
	if err != nil {
		return nil, err
	}
	return extractGroupUUIDs(iter, false)
}

// returns map with found parent uuids and originally passed uuids
func (r *GroupRepository) FindAllParentGroupsForGroupUUIDs(
	groupUUIDs map[model.GroupUUID]struct{}) (map[model.GroupUUID]struct{}, error) {
	resultGroupsSet := groupUUIDs
	currentGroupsSet := groupUUIDs
	for len(currentGroupsSet) != 0 {
		nextSet := map[model.GroupUUID]struct{}{}
		for currentGroupUUID := range currentGroupsSet {
			candidates, err := r.FindDirectParentGroupsByGroupUUID(currentGroupUUID)
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

func (r *GroupRepository) FindAllParentGroupsForUserUUID(
	userUUID model.UserUUID) (map[model.GroupUUID]struct{}, error) {
	groups, err := r.FindDirectParentGroupsByUserUUID(userUUID)
	if err != nil {
		return nil, err
	}
	return r.FindAllParentGroupsForGroupUUIDs(groups)
}

func (r *GroupRepository) FindAllParentGroupsForServiceAccountUUID(
	serviceAccountUUID model.ServiceAccountUUID) (map[model.GroupUUID]struct{}, error) {
	groups, err := r.FindDirectParentGroupsByServiceAccountUUID(serviceAccountUUID)
	if err != nil {
		return nil, err
	}
	return r.FindAllParentGroupsForGroupUUIDs(groups)
}

func (r *GroupRepository) FindAllParentGroupsForGroupUUID(
	groupUUID model.GroupUUID) (map[model.GroupUUID]struct{}, error) {
	return r.FindAllParentGroupsForGroupUUIDs(map[model.GroupUUID]struct{}{groupUUID: {}})
}

func (r *GroupRepository) FindAllMembersFor(users []model.UserUUID,
	serviceAccounts []model.ServiceAccountUUID, groups []model.GroupUUID) (map[model.UserUUID]struct{},
	map[model.ServiceAccountUUID]struct{}, error) {
	resultUsers := make(map[model.UserUUID]struct{})
	resultSAs := make(map[model.ServiceAccountUUID]struct{})

	for _, user := range users {
		resultUsers[user] = struct{}{}
	}
	for _, sa := range serviceAccounts {
		resultSAs[sa] = struct{}{}
	}

	for _, groupUUID := range groups {
		groupsUsers, groupsSAs, err := r.FindAllMembersForGroupUUID(groupUUID)
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

func (r *GroupRepository) FindAllMembersForGroupUUID(
	groupUUID model.GroupUUID) (map[model.UserUUID]struct{}, map[model.ServiceAccountUUID]struct{}, error) {
	group, err := r.GetByID(groupUUID)
	if err != nil {
		return nil, nil, err
	}
	return r.FindAllMembersForGroup(group)
}

func (r *GroupRepository) FindAllMembersForGroup(group *model.Group) (map[model.UserUUID]struct{},
	map[model.ServiceAccountUUID]struct{}, error) {
	return r.FindAllMembersFor(group.Users, group.ServiceAccounts, group.Groups)
}

func extractGroupUUIDs(iter hcmemdb.ResultIterator, showArchived bool) (map[model.GroupUUID]struct{}, error) {
	ids := map[model.GroupUUID]struct{}{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		g, ok := raw.(*model.Group)
		if !ok {
			return nil, fmt.Errorf("need type Group, actually passed: %#v", raw)
		}
		if !showArchived && g.Hash != 0 {
			continue
		}
		ids[g.UUID] = struct{}{}
	}
	return ids, nil
}
