// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type RoleBinding-parentType Tenant
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleBindingUUID = string 

const RoleBindingType = "rolebinding" // also, memdb schema name

func (u *RoleBinding) ObjType() string {
	return RoleBindingType
}

func (u *RoleBinding) ObjId() string {
	return u.UUID
}

type RoleBindingRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleBindingRepository(tx *io.MemoryStoreTxn) *RoleBindingRepository {
	return &RoleBindingRepository{db: tx}
}

func (r *RoleBindingRepository) save(rolebinding *RoleBinding) error {
	return r.db.Insert(RoleBindingType, rolebinding)
}

func (r *RoleBindingRepository) Create(rolebinding *RoleBinding) error {
	return r.save(rolebinding)
}

func (r *RoleBindingRepository) GetRawByID(id RoleBindingUUID) (interface{}, error) {
	raw, err := r.db.First(RoleBindingType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *RoleBindingRepository) GetByID(id RoleBindingUUID) (*RoleBinding, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*RoleBinding), err
}

func (r *RoleBindingRepository) Update(rolebinding *RoleBinding) error {
	_, err := r.GetByID(rolebinding.UUID)
	if err != nil {
		return err
	}
	return r.save(rolebinding)
}

func (r *RoleBindingRepository) Delete(id RoleBindingUUID) error {
	rolebinding, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(RoleBindingType, rolebinding)
}

func (r *RoleBindingRepository) List(tenantUUID TenantUUID) ([]*RoleBinding, error) {
	
	iter, err := r.db.Get(RoleBindingType, TenantForeignPK, tenantUUID)
	
	if err != nil {
		return nil, err
	}

	list := []*RoleBinding{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*RoleBinding)
		list = append(list, obj)
	}
	return list, nil
}

func (r *RoleBindingRepository) ListIDs(tenantID TenantUUID) ([]RoleBindingUUID, error) {
	objs, err := r.List(tenantID)
	if err != nil {
		return nil, err
	}
	ids := make([]RoleBindingUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *RoleBindingRepository) Iter(action func(*RoleBinding) (bool, error)) error {
	iter, err := r.db.Get(RoleBindingType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*RoleBinding)
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

func (r *RoleBindingRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	rolebinding := &RoleBinding{}
	err := json.Unmarshal(data, rolebinding)
	if err != nil {
		return err
	}

	return r.save(rolebinding)
}
