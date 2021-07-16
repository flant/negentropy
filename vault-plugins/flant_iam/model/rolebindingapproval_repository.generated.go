// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type RoleBindingApproval-parentType RoleBinding
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleBindingApprovalUUID = string 

const RoleBindingApprovalType = "role_binding_approval" // also, memdb schema name

func (u *RoleBindingApproval) ObjType() string {
	return RoleBindingApprovalType
}

func (u *RoleBindingApproval) ObjId() string {
	return u.UUID
}

type RoleBindingApprovalRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleBindingApprovalRepository(tx *io.MemoryStoreTxn) *RoleBindingApprovalRepository {
	return &RoleBindingApprovalRepository{db: tx}
}

func (r *RoleBindingApprovalRepository) save(role_binding_approval *RoleBindingApproval) error {
	return r.db.Insert(RoleBindingApprovalType, role_binding_approval)
}

func (r *RoleBindingApprovalRepository) Create(role_binding_approval *RoleBindingApproval) error {
	return r.save(role_binding_approval)
}

func (r *RoleBindingApprovalRepository) GetRawByID(id RoleBindingApprovalUUID) (interface{}, error) {
	raw, err := r.db.First(RoleBindingApprovalType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *RoleBindingApprovalRepository) GetByID(id RoleBindingApprovalUUID) (*RoleBindingApproval, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*RoleBindingApproval), err
}

func (r *RoleBindingApprovalRepository) Update(role_binding_approval *RoleBindingApproval) error {
	_, err := r.GetByID(role_binding_approval.UUID)
	if err != nil {
		return err
	}
	return r.save(role_binding_approval)
}

func (r *RoleBindingApprovalRepository) Delete(id RoleBindingApprovalUUID) error {
	role_binding_approval, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(RoleBindingApprovalType, role_binding_approval)
}

func (r *RoleBindingApprovalRepository) List(role_bindingUUID RoleBindingUUID) ([]*RoleBindingApproval, error) {
	
	iter, err := r.db.Get(RoleBindingApprovalType, RoleBindingForeignPK, role_bindingUUID)
	
	if err != nil {
		return nil, err
	}

	list := []*RoleBindingApproval{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*RoleBindingApproval)
		list = append(list, obj)
	}
	return list, nil
}

func (r *RoleBindingApprovalRepository) ListIDs(role_bindingID RoleBindingUUID) ([]RoleBindingApprovalUUID, error) {
	objs, err := r.List(role_bindingID)
	if err != nil {
		return nil, err
	}
	ids := make([]RoleBindingApprovalUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *RoleBindingApprovalRepository) Iter(action func(*RoleBindingApproval) (bool, error)) error {
	iter, err := r.db.Get(RoleBindingApprovalType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*RoleBindingApproval)
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

func (r *RoleBindingApprovalRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	role_binding_approval := &RoleBindingApproval{}
	err := json.Unmarshal(data, role_binding_approval)
	if err != nil {
		return err
	}

	return r.save(role_binding_approval)
}
