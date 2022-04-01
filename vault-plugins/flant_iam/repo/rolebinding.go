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
	RoleBindingForeignPK             = "role_binding_uuid"
	UserInRoleBindingIndex           = "user_in_role_binding"
	ServiceAccountInRoleBindingIndex = "service_account_in_role_binding"
	GroupInRoleBindingIndex          = "group_in_role_binding"
	ProjectInRoleBindingIndex        = "project_in_role_binding"
	RoleInRoleBindingIndex           = "role_in_role_binding"
	TenantUUIDRoleBindingIdIndex     = "tenant_uuid_role_binding_id"
)

func RoleBindingSchema() *memdb.DBSchema {
	var tenantUUIDRoleBindingIdIndexer []hcmemdb.Indexer

	tenantUUIDIndexer := &hcmemdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	tenantUUIDRoleBindingIdIndexer = append(tenantUUIDRoleBindingIdIndexer, tenantUUIDIndexer)

	rbIdIndexer := &hcmemdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	tenantUUIDRoleBindingIdIndexer = append(tenantUUIDRoleBindingIdIndexer, rbIdIndexer)

	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.RoleBindingType: {
				Name: model.RoleBindingType,
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
					UserInRoleBindingIndex: {
						Name:         UserInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "Users",
						},
					},
					ServiceAccountInRoleBindingIndex: {
						Name:         ServiceAccountInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "ServiceAccounts",
						},
					},
					GroupInRoleBindingIndex: {
						Name:         GroupInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "Groups",
						},
					},
					ProjectInRoleBindingIndex: {
						Name:         ProjectInRoleBindingIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer: &hcmemdb.StringSliceFieldIndex{
							Field: "Projects",
						},
					},
					RoleInRoleBindingIndex: {
						Name:         RoleInRoleBindingIndex,
						AllowMissing: false,
						Indexer: &memdb.CustomTypeSliceFieldIndexer{
							Field: "Roles",
							FromCustomType: func(customTypeValue interface{}) ([]byte, error) {
								obj, ok := customTypeValue.(model.BoundRole)
								if !ok {
									return nil, fmt.Errorf("need BoundRole, actual:%T", customTypeValue)
								}
								return []byte(obj.Name), nil
							},
						},
					},
					TenantUUIDRoleBindingIdIndex: {
						Name:    TenantUUIDRoleBindingIdIndex,
						Indexer: &hcmemdb.CompoundIndex{Indexes: tenantUUIDRoleBindingIdIndexer},
						Unique:  true,
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.RoleBindingType: {
				{OriginalDataTypeFieldName: "TenantUUID", RelatedDataType: model.TenantType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "Users", RelatedDataType: model.UserType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "Groups", RelatedDataType: model.GroupType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "ServiceAccounts", RelatedDataType: model.ServiceAccountType, RelatedDataTypeFieldIndexName: PK},
				{OriginalDataTypeFieldName: "Projects", RelatedDataType: model.ProjectType, RelatedDataTypeFieldIndexName: PK},
				{
					OriginalDataTypeFieldName: "Roles", RelatedDataType: model.RoleType, RelatedDataTypeFieldIndexName: PK,
					BuildRelatedCustomType: func(originalFieldValue interface{}) (customTypeValue interface{}, err error) {
						obj, ok := originalFieldValue.(model.BoundRole)
						if !ok {
							return nil, fmt.Errorf("need BoundRole, actual:%T", originalFieldValue)
						}
						return obj.Name, nil
					},
				},
			},
		},
		CascadeDeletes: map[string][]memdb.Relation{
			model.RoleBindingType: {{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingApprovalType, RelatedDataTypeFieldIndexName: RoleBindingForeignPK}},
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
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *RoleBindingRepository) GetByID(id model.RoleBindingUUID) (*model.RoleBinding, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	rb := raw.(*model.RoleBinding)
	if rb.FixMembers() {
		err = r.save(rb)
	}
	return rb, err
}

func (r *RoleBindingRepository) Update(rb *model.RoleBinding) error {
	_, err := r.GetByID(rb.UUID)
	if err != nil {
		return err
	}
	return r.save(rb)
}

func (r *RoleBindingRepository) CascadeDelete(id model.RoleBindingUUID, archiveMark memdb.ArchiveMark) error {
	rb, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if rb.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.CascadeArchive(model.RoleBindingType, rb, archiveMark)
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
		return nil, consts.ErrNotFound
	}
	roleBinding := raw.(*model.RoleBinding)
	return roleBinding, nil
}

func extractRoleBindings(iter hcmemdb.ResultIterator, showArchived bool) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
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
		if !showArchived && rb.Hash != 0 {
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

func (r *RoleBindingRepository) findDirectRoleBindingsForRole(role model.RoleName) (map[model.RoleBindingUUID]*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, RoleInRoleBindingIndex, model.BoundRole{Name: role})
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

func (r *RoleBindingRepository) GetByIdentifierAtTenant(tenantUUID model.TenantUUID, identifier string) (*model.RoleBinding, error) {
	iter, err := r.db.Get(model.RoleBindingType, TenantUUIDRoleBindingIdIndex, tenantUUID, identifier)
	if err != nil {
		return nil, err
	}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.RoleBinding)
		if obj.NotArchived() {
			return obj, nil
		}
	}
	return nil, consts.ErrNotFound
}
