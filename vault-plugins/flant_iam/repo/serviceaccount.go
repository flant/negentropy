package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const (
	fullIdentifierIndex             = "full_identifier"
	TenantUUIDServiceAccountIdIndex = "tenant_uuid_service_account_id"
)

func ServiceAccountSchema() *memdb.DBSchema {
	var tenantUUIDServiceAccountIdIndexer []hcmemdb.Indexer

	tenantUUIDIndexer := &hcmemdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	tenantUUIDServiceAccountIdIndexer = append(tenantUUIDServiceAccountIdIndexer, tenantUUIDIndexer)

	groupIdIndexer := &hcmemdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	tenantUUIDServiceAccountIdIndexer = append(tenantUUIDServiceAccountIdIndexer, groupIdIndexer)

	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.ServiceAccountType: {
				Name: model.ServiceAccountType,
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
					fullIdentifierIndex: {
						Name:   fullIdentifierIndex,
						Unique: true,
						Indexer: &hcmemdb.StringFieldIndex{
							Field:     "FullIdentifier",
							Lowercase: true,
						},
					},
					TenantUUIDServiceAccountIdIndex: {
						Name:    TenantUUIDServiceAccountIdIndex,
						Indexer: &hcmemdb.CompoundIndex{Indexes: tenantUUIDServiceAccountIdIndexer},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.ServiceAccountType: {{OriginalDataTypeFieldName: "TenantUUID", RelatedDataType: model.TenantType, RelatedDataTypeFieldIndexName: PK}},
		},
		CascadeDeletes: map[string][]memdb.Relation{
			model.ServiceAccountType: {
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.GroupType, RelatedDataTypeFieldIndexName: ServiceAccountInGroupIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingType, RelatedDataTypeFieldIndexName: ServiceAccountInRoleBindingIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.RoleBindingApprovalType, RelatedDataTypeFieldIndexName: ServiceAccountInRoleBindingApprovalIndex},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.MultipassType, RelatedDataTypeFieldIndexName: OwnerForeignPK},
				{OriginalDataTypeFieldName: "UUID", RelatedDataType: model.ServiceAccountPasswordType, RelatedDataTypeFieldIndexName: OwnerForeignPK},
			},
		},
	}
}

type ServiceAccountRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewServiceAccountRepository(tx *io.MemoryStoreTxn) *ServiceAccountRepository {
	return &ServiceAccountRepository{db: tx}
}

func (r *ServiceAccountRepository) save(sa *model.ServiceAccount) error {
	return r.db.Insert(model.ServiceAccountType, sa)
}

func (r *ServiceAccountRepository) Create(sa *model.ServiceAccount) error {
	return r.save(sa)
}

func (r *ServiceAccountRepository) GetRawByID(id model.ServiceAccountUUID) (interface{}, error) {
	raw, err := r.db.First(model.ServiceAccountType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *ServiceAccountRepository) GetByID(id model.ServiceAccountUUID) (*model.ServiceAccount, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.ServiceAccount), err
}

func (r *ServiceAccountRepository) Update(sa *model.ServiceAccount) error {
	_, err := r.GetByID(sa.UUID)
	if err != nil {
		return err
	}
	return r.save(sa)
}

func (r *ServiceAccountRepository) CascadeDelete(id model.ServiceAccountUUID,
	archiveMark memdb.ArchiveMark) error {
	sa, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if sa.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.CascadeArchive(model.ServiceAccountType, sa, archiveMark)
}

func (r *ServiceAccountRepository) CleanChildrenSliceIndexes(id model.ServiceAccountUUID) error {
	sa, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.CleanChildrenSliceIndexes(model.ServiceAccountType, sa)
}

func (r *ServiceAccountRepository) CascadeRestore(id model.ServiceAccountUUID) (*model.ServiceAccount, error) {
	sa, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !sa.Archived() {
		return nil, consts.ErrIsNotArchived
	}
	err = r.db.CascadeRestore(model.UserType, sa)
	if err != nil {
		return nil, err
	}
	return sa, nil
}

func (r *ServiceAccountRepository) List(tenantUUID model.TenantUUID, showArchived bool) ([]*model.ServiceAccount, error) {
	iter, err := r.db.Get(model.ServiceAccountType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.ServiceAccount{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.ServiceAccount)
		if showArchived || obj.Timestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *ServiceAccountRepository) ListIDs(tenantID model.TenantUUID, showArchived bool) ([]model.ServiceAccountUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.ServiceAccountUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ServiceAccountRepository) Iter(action func(*model.ServiceAccount) (bool, error)) error {
	iter, err := r.db.Get(model.ServiceAccountType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.ServiceAccount)
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

func (r *ServiceAccountRepository) Sync(_ string, data []byte) error {
	sa := &model.ServiceAccount{}
	err := json.Unmarshal(data, sa)
	if err != nil {
		return err
	}

	return r.save(sa)
}

func (r *ServiceAccountRepository) GetByIdentifier(tenantUUID, identifier string) (*model.ServiceAccount, error) {
	raw, err := r.db.First(model.ServiceAccountType, TenantUUIDServiceAccountIdIndex, tenantUUID, identifier)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw.(*model.ServiceAccount), err
}

// TODO move to usecases
// generic: <identifier>@serviceaccount.<tenant_identifier>
func CalcServiceAccountFullIdentifier(saID, tenantID string) string {
	domain := "serviceaccount." + tenantID

	return saID + "@" + domain
}
