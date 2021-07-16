// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type Group-parentType Tenant
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type GroupUUID = string 

const GroupType = "group" // also, memdb schema name

func (u *Group) ObjType() string {
	return GroupType
}

func (u *Group) ObjId() string {
	return u.UUID
}

type GroupRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewGroupRepository(tx *io.MemoryStoreTxn) *GroupRepository {
	return &GroupRepository{db: tx}
}

func (r *GroupRepository) save(group *Group) error {
	return r.db.Insert(GroupType, group)
}

func (r *GroupRepository) Create(group *Group) error {
	return r.save(group)
}

func (r *GroupRepository) GetRawByID(id GroupUUID) (interface{}, error) {
	raw, err := r.db.First(GroupType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *GroupRepository) GetByID(id GroupUUID) (*Group, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*Group), err
}

func (r *GroupRepository) Update(group *Group) error {
	_, err := r.GetByID(group.UUID)
	if err != nil {
		return err
	}
	return r.save(group)
}

func (r *GroupRepository) Delete(id GroupUUID) error {
	group, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(GroupType, group)
}

func (r *GroupRepository) List(tenantUUID TenantUUID) ([]*Group, error) {
	
	iter, err := r.db.Get(GroupType, TenantForeignPK, tenantUUID)
	
	if err != nil {
		return nil, err
	}

	list := []*Group{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Group)
		list = append(list, obj)
	}
	return list, nil
}

func (r *GroupRepository) ListIDs(tenantID TenantUUID) ([]GroupUUID, error) {
	objs, err := r.List(tenantID)
	if err != nil {
		return nil, err
	}
	ids := make([]GroupUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *GroupRepository) Iter(action func(*Group) (bool, error)) error {
	iter, err := r.db.Get(GroupType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Group)
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

func (r *GroupRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	group := &Group{}
	err := json.Unmarshal(data, group)
	if err != nil {
		return err
	}

	return r.save(group)
}
