// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type IdentitySharing-parentType Tenant
// When: 2021-07-16 14:19:24.399948 +0300 MSK m=+0.000132986
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type IdentitySharingUUID = string 

const IdentitySharingType = "identitysharing" // also, memdb schema name

func (u *IdentitySharing) ObjType() string {
	return IdentitySharingType
}

func (u *IdentitySharing) ObjId() string {
	return u.UUID
}

type IdentitySharingRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewIdentitySharingRepository(tx *io.MemoryStoreTxn) *IdentitySharingRepository {
	return &IdentitySharingRepository{db: tx}
}

func (r *IdentitySharingRepository) save(identitysharing *IdentitySharing) error {
	return r.db.Insert(IdentitySharingType, identitysharing)
}

func (r *IdentitySharingRepository) Create(identitysharing *IdentitySharing) error {
	return r.save(identitysharing)
}

func (r *IdentitySharingRepository) GetRawByID(id IdentitySharingUUID) (interface{}, error) {
	raw, err := r.db.First(IdentitySharingType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *IdentitySharingRepository) GetByID(id IdentitySharingUUID) (*IdentitySharing, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*IdentitySharing), err
}

func (r *IdentitySharingRepository) Update(identitysharing *IdentitySharing) error {
	_, err := r.GetByID(identitysharing.UUID)
	if err != nil {
		return err
	}
	return r.save(identitysharing)
}

func (r *IdentitySharingRepository) Delete(id IdentitySharingUUID) error {
	identitysharing, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(IdentitySharingType, identitysharing)
}

func (r *IdentitySharingRepository) List(tenantUUID TenantUUID) ([]*IdentitySharing, error) {
	
	iter, err := r.db.Get(IdentitySharingType, TenantForeignPK, tenantUUID)
	
	if err != nil {
		return nil, err
	}

	list := []*IdentitySharing{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*IdentitySharing)
		list = append(list, obj)
	}
	return list, nil
}

func (r *IdentitySharingRepository) ListIDs(tenantID TenantUUID) ([]IdentitySharingUUID, error) {
	objs, err := r.List(tenantID)
	if err != nil {
		return nil, err
	}
	ids := make([]IdentitySharingUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *IdentitySharingRepository) Iter(action func(*IdentitySharing) (bool, error)) error {
	iter, err := r.db.Get(IdentitySharingType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*IdentitySharing)
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

func (r *IdentitySharingRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	identitysharing := &IdentitySharing{}
	err := json.Unmarshal(data, identitysharing)
	if err != nil {
		return err
	}

	return r.save(identitysharing)
}
