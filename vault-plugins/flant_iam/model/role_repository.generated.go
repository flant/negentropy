// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type Role
// When: 2021-07-16 14:19:25.499922 +0300 MSK m=+0.000137644
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleName = string 

const RoleType = "role" // also, memdb schema name

func (u *Role) ObjType() string {
	return RoleType
}

func (u *Role) ObjId() string {
	return u.Name
}

type RoleRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewRoleRepository(tx *io.MemoryStoreTxn) *RoleRepository {
	return &RoleRepository{db: tx}
}

func (r *RoleRepository) save(role *Role) error {
	return r.db.Insert(RoleType, role)
}

func (r *RoleRepository) Create(role *Role) error {
	return r.save(role)
}

func (r *RoleRepository) GetRawByID(id RoleName) (interface{}, error) {
	raw, err := r.db.First(RoleType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *RoleRepository) GetByID(id RoleName) (*Role, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*Role), err
}

func (r *RoleRepository) Update(role *Role) error {
	_, err := r.GetByID(role.Name)
	if err != nil {
		return err
	}
	return r.save(role)
}

func (r *RoleRepository) Delete(id RoleName) error {
	role, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(RoleType, role)
}

func (r *RoleRepository) List() ([]*Role, error) {
	
	iter, err := r.db.Get(RoleType, PK)
	
	if err != nil {
		return nil, err
	}

	list := []*Role{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Role)
		list = append(list, obj)
	}
	return list, nil
}

func (r *RoleRepository) ListIDs() ([]RoleName, error) {
	objs, err := r.List()
	if err != nil {
		return nil, err
	}
	ids := make([]RoleName, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *RoleRepository) Iter(action func(*Role) (bool, error)) error {
	iter, err := r.db.Get(RoleType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Role)
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

func (r *RoleRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	role := &Role{}
	err := json.Unmarshal(data, role)
	if err != nil {
		return err
	}

	return r.save(role)
}
