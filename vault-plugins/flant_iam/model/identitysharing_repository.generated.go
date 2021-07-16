// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type IdentitySharing-parentType Tenant
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type IdentitySharingUUID = string 

const IdentitySharingType = "identity_sharing" // also, memdb schema name

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

func (r *IdentitySharingRepository) save(identity_sharing *IdentitySharing) error {
	return r.db.Insert(IdentitySharingType, identity_sharing)
}

func (r *IdentitySharingRepository) Create(identity_sharing *IdentitySharing) error {
	return r.save(identity_sharing)
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

func (r *IdentitySharingRepository) Update(identity_sharing *IdentitySharing) error {
	_, err := r.GetByID(identity_sharing.UUID)
	if err != nil {
		return err
	}
	return r.save(identity_sharing)
}

func (r *IdentitySharingRepository) Delete(id IdentitySharingUUID) error {
	identity_sharing, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(IdentitySharingType, identity_sharing)
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

	identity_sharing := &IdentitySharing{}
	err := json.Unmarshal(data, identity_sharing)
	if err != nil {
		return err
	}

	return r.save(identity_sharing)
}
