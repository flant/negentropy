// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type User-parentType Tenant
// When: 2021-07-16 14:19:27.669228 +0300 MSK m=+0.000136690
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type UserUUID = string 

const UserType = "user" // also, memdb schema name

func (u *User) ObjType() string {
	return UserType
}

func (u *User) ObjId() string {
	return u.UUID
}

type UserRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewUserRepository(tx *io.MemoryStoreTxn) *UserRepository {
	return &UserRepository{db: tx}
}

func (r *UserRepository) save(user *User) error {
	return r.db.Insert(UserType, user)
}

func (r *UserRepository) Create(user *User) error {
	return r.save(user)
}

func (r *UserRepository) GetRawByID(id UserUUID) (interface{}, error) {
	raw, err := r.db.First(UserType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *UserRepository) GetByID(id UserUUID) (*User, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*User), err
}

func (r *UserRepository) Update(user *User) error {
	_, err := r.GetByID(user.UUID)
	if err != nil {
		return err
	}
	return r.save(user)
}

func (r *UserRepository) Delete(id UserUUID) error {
	user, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(UserType, user)
}

func (r *UserRepository) List(tenantUUID TenantUUID) ([]*User, error) {
	
	iter, err := r.db.Get(UserType, TenantForeignPK, tenantUUID)
	
	if err != nil {
		return nil, err
	}

	list := []*User{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*User)
		list = append(list, obj)
	}
	return list, nil
}

func (r *UserRepository) ListIDs(tenantID TenantUUID) ([]UserUUID, error) {
	objs, err := r.List(tenantID)
	if err != nil {
		return nil, err
	}
	ids := make([]UserUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *UserRepository) Iter(action func(*User) (bool, error)) error {
	iter, err := r.db.Get(UserType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*User)
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

func (r *UserRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	user := &User{}
	err := json.Unmarshal(data, user)
	if err != nil {
		return err
	}

	return r.save(user)
}
