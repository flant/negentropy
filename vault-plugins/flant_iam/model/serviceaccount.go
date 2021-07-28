package model

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	fullIdentifierIndex             = "full_identifier"
	TenantUUIDServiceAccountIdIndex = "tenant_uuid_service_account_id"
)

type ServiceAccountObjectType string

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
			ServiceAccountType: {
				Name: ServiceAccountType,
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

type ServiceAccount struct {
	UUID           ServiceAccountUUID `json:"uuid"` // PK
	TenantUUID     TenantUUID         `json:"tenant_uuid"`
	Version        string             `json:"resource_version"`
	BuiltinType    string             `json:"-"`
	Identifier     string             `json:"identifier"`
	FullIdentifier string             `json:"full_identifier"`
	CIDRs          []string           `json:"allowed_cidrs"`
	TokenTTL       time.Duration      `json:"token_ttl"`
	TokenMaxTTL    time.Duration      `json:"token_max_ttl"`

	Origin ObjectOrigin `json:"origin"`

	Extensions         map[ObjectOrigin]*Extension `json:"-"`
	ArchivingTimestamp UnixTime                    `json:"archiving_timestamp"`
	ArchivingHash      int64                       `json:"archiving_hash"`
}

const ServiceAccountType = "service_account" // also, memdb schema name

func (u *ServiceAccount) ObjType() string {
	return ServiceAccountType
}

func (u *ServiceAccount) ObjId() string {
	return u.UUID
}

type ServiceAccountRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewServiceAccountRepository(tx *io.MemoryStoreTxn) *ServiceAccountRepository {
	return &ServiceAccountRepository{db: tx}
}

func (r *ServiceAccountRepository) save(sa *ServiceAccount) error {
	return r.db.Insert(ServiceAccountType, sa)
}

func (r *ServiceAccountRepository) Create(sa *ServiceAccount) error {
	return r.save(sa)
}

func (r *ServiceAccountRepository) GetRawByID(id ServiceAccountUUID) (interface{}, error) {
	raw, err := r.db.First(ServiceAccountType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *ServiceAccountRepository) GetByID(id ServiceAccountUUID) (*ServiceAccount, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*ServiceAccount), err
}

func (r *ServiceAccountRepository) Update(sa *ServiceAccount) error {
	_, err := r.GetByID(sa.UUID)
	if err != nil {
		return err
	}
	return r.save(sa)
}

func (r *ServiceAccountRepository) Delete(id ServiceAccountUUID,
	archivingTimestamp UnixTime, archivingHash int64) error {
	sa, err := r.GetByID(id)
	if err != nil {
		return err
	}
	sa.ArchivingTimestamp = archivingTimestamp
	sa.ArchivingHash = archivingHash
	return r.Update(sa)
}

func (r *ServiceAccountRepository) List(tenantUUID TenantUUID, showArchived bool) ([]*ServiceAccount, error) {
	iter, err := r.db.Get(ServiceAccountType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*ServiceAccount{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*ServiceAccount)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *ServiceAccountRepository) ListIDs(tenantID TenantUUID, showArchived bool) ([]ServiceAccountUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]ServiceAccountUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ServiceAccountRepository) Iter(action func(*ServiceAccount) (bool, error)) error {
	iter, err := r.db.Get(ServiceAccountType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*ServiceAccount)
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
	sa := &ServiceAccount{}
	err := json.Unmarshal(data, sa)
	if err != nil {
		return err
	}

	return r.save(sa)
}

func (r *ServiceAccountRepository) GetByIdentifier(tenantUUID, identifier string) (*ServiceAccount, error) {
	raw, err := r.db.First(ServiceAccountType, TenantUUIDServiceAccountIdIndex, tenantUUID, identifier)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*ServiceAccount), err
}

// TODO move to usecases
// generic: <identifier>@serviceaccount.<tenant_identifier>
func CalcServiceAccountFullIdentifier(saID, tenantID string) string {
	domain := "serviceaccount." + tenantID

	return saID + "@" + domain
}
