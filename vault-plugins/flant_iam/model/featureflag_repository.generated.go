// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type FeatureFlag
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type FeatureFlagName = string 

const FeatureFlagType = "featureflag" // also, memdb schema name

func (u *FeatureFlag) ObjType() string {
	return FeatureFlagType
}

func (u *FeatureFlag) ObjId() string {
	return u.Name
}

type FeatureFlagRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewFeatureFlagRepository(tx *io.MemoryStoreTxn) *FeatureFlagRepository {
	return &FeatureFlagRepository{db: tx}
}

func (r *FeatureFlagRepository) save(featureflag *FeatureFlag) error {
	return r.db.Insert(FeatureFlagType, featureflag)
}

func (r *FeatureFlagRepository) Create(featureflag *FeatureFlag) error {
	return r.save(featureflag)
}

func (r *FeatureFlagRepository) GetRawByID(id FeatureFlagName) (interface{}, error) {
	raw, err := r.db.First(FeatureFlagType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *FeatureFlagRepository) GetByID(id FeatureFlagName) (*FeatureFlag, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*FeatureFlag), err
}

func (r *FeatureFlagRepository) Update(featureflag *FeatureFlag) error {
	_, err := r.GetByID(featureflag.Name)
	if err != nil {
		return err
	}
	return r.save(featureflag)
}

func (r *FeatureFlagRepository) Delete(id FeatureFlagName) error {
	featureflag, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(FeatureFlagType, featureflag)
}

func (r *FeatureFlagRepository) List() ([]*FeatureFlag, error) {
	
	iter, err := r.db.Get(FeatureFlagType, PK)
	
	if err != nil {
		return nil, err
	}

	list := []*FeatureFlag{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*FeatureFlag)
		list = append(list, obj)
	}
	return list, nil
}

func (r *FeatureFlagRepository) ListIDs() ([]FeatureFlagName, error) {
	objs, err := r.List()
	if err != nil {
		return nil, err
	}
	ids := make([]FeatureFlagName, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *FeatureFlagRepository) Iter(action func(*FeatureFlag) (bool, error)) error {
	iter, err := r.db.Get(FeatureFlagType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*FeatureFlag)
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

func (r *FeatureFlagRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	featureflag := &FeatureFlag{}
	err := json.Unmarshal(data, featureflag)
	if err != nil {
		return err
	}

	return r.save(featureflag)
}
