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
	TenantForeignPK          = "tenant_uuid"
	FeatureFlagInTenantIndex = "feature_flag_in_tenant"
	TenantIdentifier         = "identifier"
)

func TenantSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.TenantType: {
				Name: model.TenantType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantIdentifier: {
						Name: TenantIdentifier,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "Identifier",
							Lowercase: true,
						},
					},
					"version": {
						Name: "version",
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "Version",
						},
					},
					FeatureFlagInTenantIndex: {
						Name:         FeatureFlagInTenantIndex,
						AllowMissing: true,
						Indexer: &memdb.CustomTypeSliceFieldIndexer{
							Field: "FeatureFlags",
							FromCustomType: func(customTypeValue interface{}) ([]byte, error) {
								obj, ok := customTypeValue.(model.TenantFeatureFlag)
								if !ok {
									return nil, fmt.Errorf("need TenantFeatureFlag, actual:%T", customTypeValue)
								}
								return []byte(obj.Name), nil
							},
						},
					},
				},
			},
		},
		CascadeDeletes: map[memdb.DataType][]memdb.Relation{
			model.TenantType: {
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.UserType, RelatedDataTypeFieldIndexName: TenantForeignPK},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.IdentitySharingType, RelatedDataTypeFieldIndexName: DestinationTenantUUIDIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingApprovalType, RelatedDataTypeFieldIndexName: TenantForeignPK},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.ServiceAccountPasswordType, RelatedDataTypeFieldIndexName: TenantForeignPK},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.ServiceAccountType, RelatedDataTypeFieldIndexName: TenantForeignPK},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.GroupType, RelatedDataTypeFieldIndexName: TenantForeignPK},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.ProjectType, RelatedDataTypeFieldIndexName: TenantForeignPK},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.MultipassType, RelatedDataTypeFieldIndexName: TenantForeignPK},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingType, RelatedDataTypeFieldIndexName: TenantForeignPK},
			},
		},
		CheckingRelations: map[memdb.DataType][]memdb.Relation{
			model.TenantType: {{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.IdentitySharingType, RelatedDataTypeFieldIndexName: SourceTenantUUIDIndex}},
		},
		UniqueConstraints: map[memdb.DataType][]memdb.IndexName{
			model.TenantType: {TenantIdentifier},
		},
	}
}

type TenantRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewTenantRepository(tx *io.MemoryStoreTxn) *TenantRepository {
	return &TenantRepository{db: tx}
}

func (r *TenantRepository) save(tenant *model.Tenant) error {
	return r.db.Insert(model.TenantType, tenant)
}

func (r *TenantRepository) Create(tenant *model.Tenant) error {
	return r.save(tenant)
}

func (r *TenantRepository) GetRawByID(id model.TenantUUID) (interface{}, error) {
	raw, err := r.db.First(model.TenantType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *TenantRepository) GetByID(id model.TenantUUID) (*model.Tenant, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Tenant), err
}

func (r *TenantRepository) Update(tenant *model.Tenant) error {
	_, err := r.GetByID(tenant.UUID)
	if err != nil {
		return err
	}
	return r.save(tenant)
}

func (r *TenantRepository) CascadeDelete(id model.TenantUUID, archiveMark memdb.ArchiveMark) error {
	tenant, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if tenant.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.CascadeArchive(model.TenantType, tenant, archiveMark)
}

func (r *TenantRepository) List(showArchived bool) ([]*model.Tenant, error) {
	iter, err := r.db.Get(model.TenantType, PK)
	if err != nil {
		return nil, err
	}

	list := []*model.Tenant{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Tenant)
		if showArchived || obj.NotArchived() {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *TenantRepository) ListIDs(showArchived bool) ([]model.TenantUUID, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.TenantUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *TenantRepository) Iter(action func(*model.Tenant) (bool, error)) error {
	iter, err := r.db.Get(model.TenantType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Tenant)
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

func (r *TenantRepository) Sync(_ string, data []byte) error {
	tenant := &model.Tenant{}
	err := json.Unmarshal(data, tenant)
	if err != nil {
		return err
	}

	return r.save(tenant)
}

func (r *TenantRepository) Restore(id model.TenantUUID) (*model.Tenant, error) {
	tenant, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if tenant.NotArchived() {
		return nil, consts.ErrIsNotArchived
	}
	err = r.db.Restore(model.TenantType, tenant)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

func (r *TenantRepository) CascadeRestore(id model.TenantUUID) (*model.Tenant, error) {
	tenant, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if tenant.NotArchived() {
		return nil, consts.ErrIsNotArchived
	}
	err = r.db.CascadeRestore(model.TenantType, tenant)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

func (r *TenantRepository) GetByIdentifier(identifier string) (*model.Tenant, error) {
	raw, err := r.db.First(model.TenantType, TenantIdentifier, identifier)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw.(*model.Tenant), err
}

func (r *TenantRepository) CascadeErase(id model.TenantUUID) error {
	tenant, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if tenant.NotArchived() {
		return consts.ErrIsNotArchived
	}
	return r.db.CascadeDelete(model.TenantType, tenant)
}
