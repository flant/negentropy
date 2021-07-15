package model

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	ServiceAccountType = "service_account" // also, memdb schema name

)

type ServiceAccountObjectType string

func ServiceAccountSchema() *memdb.DBSchema {
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

	Extensions map[ObjectOrigin]*Extension `json:"-"`
}

func (sa *ServiceAccount) ObjType() string {
	return ServiceAccountType
}

func (sa *ServiceAccount) ObjId() string {
	return sa.UUID
}

// generic: <identifier>@serviceaccount.<tenant_identifier>
// builtin: <identifier>@<builtin_service_account_type>.serviceaccount.<tenant_identifier>
func CalcServiceAccountFullIdentifier(sa *ServiceAccount, tenant *Tenant) string {
	name := sa.Identifier
	domain := "serviceaccount." + tenant.Identifier
	if sa.BuiltinType != "" {
		domain = sa.BuiltinType + "." + domain
	}
	return name + "@" + domain
}

type ServiceAccountRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewServiceAccountRepository(tx *io.MemoryStoreTxn) *ServiceAccountRepository {
	return &ServiceAccountRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *ServiceAccountRepository) save(sa *ServiceAccount) error {
	return r.db.Insert(ServiceAccountType, sa)
}

func (r *ServiceAccountRepository) Create(sa *ServiceAccount) error {
	return r.save(sa)
}

func (r *ServiceAccountRepository) GetByID(id ServiceAccountUUID) (*ServiceAccount, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*ServiceAccount), err
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

func (r *ServiceAccountRepository) GetByIdentifier(said, tid string) (*ServiceAccount, error) {
	// TODO move the calculation to usecases, accept only prepared fullID
	fullID := CalcServiceAccountFullIdentifier(said, tid)

	raw, err := r.db.First(ServiceAccountType, fullIdentifierIndex, fullID)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	serviceAccount := raw.(*ServiceAccount)
	return serviceAccount, nil
}

func (r *ServiceAccountRepository) Update(sa *ServiceAccount) error {
	_, err := r.GetByID(sa.UUID)
	if err != nil {
		return err
	}

	return r.save(sa)
}

func (r *ServiceAccountRepository) Delete(id ServiceAccountUUID) error {
	sa, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(ServiceAccountType, sa)
}

func (r *ServiceAccountRepository) List(tenantID TenantUUID) ([]*ServiceAccount, error) {
	iter, err := r.db.Get(ServiceAccountType, TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	list := []*ServiceAccount{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		sa := raw.(*ServiceAccount)
		list = append(list, sa)
	}
	return list, nil
}

func (r *ServiceAccountRepository) Iter(action func(account *ServiceAccount) (bool, error)) error {
	iter, err := r.db.Get(ServiceAccountType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*ServiceAccount)
		next, err := action(t)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
}

func (r *ServiceAccountRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	sa := &ServiceAccount{}
	err := json.Unmarshal(data, sa)
	if err != nil {
		return err
	}

	return r.save(sa)
}

// TODO move to usecases
// generic: <identifier>@serviceaccount.<tenant_identifier>
func CalcServiceAccountFullIdentifier(saID, tenantID string) string {
	domain := "serviceaccount." + tenantID

	return saID + "@" + domain
}
