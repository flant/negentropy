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
	tenant, err := r.tenantRepo.GetByID(sa.TenantUUID)
	if err != nil {
		return err
	}
	if sa.Version != "" {
		return ErrBadVersion
	}
	if sa.Origin == "" {
		return ErrBadOrigin
	}
	sa.Version = NewResourceVersion()
	sa.FullIdentifier = CalcServiceAccountFullIdentifier(sa, tenant)

	return r.save(sa)
}

func (r *ServiceAccountRepository) GetByID(id ServiceAccountUUID) (*ServiceAccount, error) {
	raw, err := r.GetRawByID(id)
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

/*
TODO
	* Из-за того, что в очереди формата TokenGenerationNumber стоит ttl 30 дней – token_ttl не может быть больше 30 дней.
		См. подробнее следующий пункт и описание формата очереди.

TODO Логика создания/обновления сервис аккаунта:
	* type object_with_resource_version
	* type tenanted_object
	* validate_tenant(запрос, объект из стора)
	* validate_resource_version(запрос, entry)
	* пробуем загрузить объект, если объект есть, то:
	* валидируем resource_version
	* валидируем тенанта
	* валидируем builtin_type_name
	* если объекта нет, то:
	* валидируем, что нам не передан resource_version
*/
func (r *ServiceAccountRepository) Update(sa *ServiceAccount) error {
	stored, err := r.GetByID(sa.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != sa.TenantUUID {
		return ErrNotFound
	}
	if stored.Origin != sa.Origin {
		return ErrBadOrigin
	}
	if stored.Version != sa.Version {
		return ErrBadVersion
	}
	sa.Version = NewResourceVersion()

	// Update

	tenant, err := r.tenantRepo.GetByID(sa.TenantUUID)
	if err != nil {
		return err
	}
	sa.FullIdentifier = CalcServiceAccountFullIdentifier(sa, tenant)

	// Preserve fields, that are not always accessable from the outside, e.g. from HTTP API
	if sa.Extensions == nil {
		sa.Extensions = stored.Extensions
	}

	return r.save(sa)
}

/*
TODO
	* При удалении необходимо удалить все “вложенные” объекты (Token и ServiceAccountPassword).
	* При удалении необходимо удалить из всех связей (из групп, из role_binding’ов, из approval’ов и пр.)
*/
func (r *ServiceAccountRepository) delete(id ServiceAccountUUID) error {
	sa, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(ServiceAccountType, sa)
}

func (r *ServiceAccountRepository) Delete(origin ObjectOrigin, id ServiceAccountUUID) error {
	sa, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if sa.Origin != origin {
		return ErrBadOrigin
	}
	return r.delete(id)
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

func (r *ServiceAccountRepository) DeleteByTenant(tenantUUID TenantUUID) error {
	_, err := r.db.DeleteAll(ServiceAccountType, TenantForeignPK, tenantUUID)
	return err
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

func (r *ServiceAccountRepository) SetExtension(ext *Extension) error {
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

func (r *ServiceAccountRepository) UnsetExtension(origin ObjectOrigin, uuid string) error {
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

func (r *ServiceAccountRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.delete(objID)
	}

	sa := &ServiceAccount{}
	err := json.Unmarshal(data, sa)
	if err != nil {
		return err
	}

	return r.save(sa)
}
