// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type ServiceAccount-parentType Tenant
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ServiceAccountUUID = string 

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

func (r *ServiceAccountRepository) save(service_account *ServiceAccount) error {
	return r.db.Insert(ServiceAccountType, service_account)
}

func (r *ServiceAccountRepository) Create(service_account *ServiceAccount) error {
	return r.save(service_account)
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

func (r *ServiceAccountRepository) Update(service_account *ServiceAccount) error {
	_, err := r.GetByID(service_account.UUID)
	if err != nil {
		return err
	}
	return r.save(service_account)
}

func (r *ServiceAccountRepository) Delete(id ServiceAccountUUID) error {
	service_account, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(ServiceAccountType, service_account)
}

func (r *ServiceAccountRepository) List(tenantUUID TenantUUID) ([]*ServiceAccount, error) {
	
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
		list = append(list, obj)
	}
	return list, nil
}

func (r *ServiceAccountRepository) ListIDs(tenantID TenantUUID) ([]ServiceAccountUUID, error) {
	objs, err := r.List(tenantID)
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

func (r *ServiceAccountRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	service_account := &ServiceAccount{}
	err := json.Unmarshal(data, service_account)
	if err != nil {
		return err
	}

	return r.save(service_account)
}
