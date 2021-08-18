package repo

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	fullIdentifierIndex             = "full_identifier"
	TenantUUIDServiceAccountIdIndex = "tenant_uuid_service_account_id"
)

func ServiceAccountSchema() *memdb.DBSchema {
	var tenantUUIDServiceAccountIdIndexer []memdb.Indexer

	tenantUUIDIndexer := &memdb.StringFieldIndex{
		Field:     "TenantUUID",
		Lowercase: true,
	}
	tenantUUIDServiceAccountIdIndexer = append(tenantUUIDServiceAccountIdIndexer, tenantUUIDIndexer)

	groupIdIndexer := &memdb.StringFieldIndex{
		Field:     "Identifier",
		Lowercase: true,
	}
	tenantUUIDServiceAccountIdIndexer = append(tenantUUIDServiceAccountIdIndexer, groupIdIndexer)

	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			model.ServiceAccountType: {
				Name: model.ServiceAccountType,
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
					fullIdentifierIndex: {
						Name: fullIdentifierIndex,
						Indexer: &memdb.StringFieldIndex{
							Field:     "FullIdentifier",
							Lowercase: true,
						},
					},
					TenantUUIDServiceAccountIdIndex: {
						Name:    TenantUUIDServiceAccountIdIndex,
						Indexer: &memdb.CompoundIndex{Indexes: tenantUUIDServiceAccountIdIndexer},
					},
				},
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
		return nil, model.ErrNotFound
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

func (r *ServiceAccountRepository) Delete(id model.ServiceAccountUUID,
	archivingTimestamp model.UnixTime, archivingHash int64) error {
	sa, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if sa.IsDeleted() {
		return model.ErrIsArchived
	}
	sa.ArchivingTimestamp = archivingTimestamp
	sa.ArchivingHash = archivingHash
	return r.Update(sa)
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
		if showArchived || obj.ArchivingTimestamp == 0 {
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
		return nil, model.ErrNotFound
	}
	return raw.(*model.ServiceAccount), err
}

// TODO move to usecases
// generic: <identifier>@serviceaccount.<tenant_identifier>
func CalcServiceAccountFullIdentifier(saID, tenantID string) string {
	domain := "serviceaccount." + tenantID

	return saID + "@" + domain
}
