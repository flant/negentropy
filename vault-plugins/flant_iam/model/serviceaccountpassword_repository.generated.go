// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type ServiceAccountPassword-parentType Owner
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ServiceAccountPasswordUUID = string 

const ServiceAccountPasswordType = "service_account_password" // also, memdb schema name

func (u *ServiceAccountPassword) ObjType() string {
	return ServiceAccountPasswordType
}

func (u *ServiceAccountPassword) ObjId() string {
	return u.UUID
}

type ServiceAccountPasswordRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewServiceAccountPasswordRepository(tx *io.MemoryStoreTxn) *ServiceAccountPasswordRepository {
	return &ServiceAccountPasswordRepository{db: tx}
}

func (r *ServiceAccountPasswordRepository) save(service_account_password *ServiceAccountPassword) error {
	return r.db.Insert(ServiceAccountPasswordType, service_account_password)
}

func (r *ServiceAccountPasswordRepository) Create(service_account_password *ServiceAccountPassword) error {
	return r.save(service_account_password)
}

func (r *ServiceAccountPasswordRepository) GetRawByID(id ServiceAccountPasswordUUID) (interface{}, error) {
	raw, err := r.db.First(ServiceAccountPasswordType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *ServiceAccountPasswordRepository) GetByID(id ServiceAccountPasswordUUID) (*ServiceAccountPassword, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*ServiceAccountPassword), err
}

func (r *ServiceAccountPasswordRepository) Update(service_account_password *ServiceAccountPassword) error {
	_, err := r.GetByID(service_account_password.UUID)
	if err != nil {
		return err
	}
	return r.save(service_account_password)
}

func (r *ServiceAccountPasswordRepository) Delete(id ServiceAccountPasswordUUID) error {
	service_account_password, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(ServiceAccountPasswordType, service_account_password)
}

func (r *ServiceAccountPasswordRepository) List(ownerUUID OwnerUUID) ([]*ServiceAccountPassword, error) {
	
	iter, err := r.db.Get(ServiceAccountPasswordType, OwnerForeignPK, ownerUUID)
	
	if err != nil {
		return nil, err
	}

	list := []*ServiceAccountPassword{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*ServiceAccountPassword)
		list = append(list, obj)
	}
	return list, nil
}

func (r *ServiceAccountPasswordRepository) ListIDs(ownerID OwnerUUID) ([]ServiceAccountPasswordUUID, error) {
	objs, err := r.List(ownerID)
	if err != nil {
		return nil, err
	}
	ids := make([]ServiceAccountPasswordUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ServiceAccountPasswordRepository) Iter(action func(*ServiceAccountPassword) (bool, error)) error {
	iter, err := r.db.Get(ServiceAccountPasswordType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*ServiceAccountPassword)
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

func (r *ServiceAccountPasswordRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	service_account_password := &ServiceAccountPassword{}
	err := json.Unmarshal(data, service_account_password)
	if err != nil {
		return err
	}

	return r.save(service_account_password)
}
