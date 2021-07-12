package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	FeatureFlagType = "feature_flag" // also, memdb schema name
)

func FeatureFlagSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			FeatureFlagType: {
				Name: FeatureFlagType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.StringFieldIndex{
							Field: "Name",
						},
					},
				},
			},
		},
	}
}

type FeatureFlag struct {
	Name FeatureFlagName `json:"name"` // PK
}

func (t *FeatureFlag) ObjType() string {
	return FeatureFlagType
}

func (t *FeatureFlag) ObjId() string {
	return t.Name
}

type FeatureFlagRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewFeatureFlagRepository(tx *io.MemoryStoreTxn) *FeatureFlagRepository {
	return &FeatureFlagRepository{tx}
}

func (r *FeatureFlagRepository) Create(ff *FeatureFlag) error {
	_, err := r.Get(ff.Name)
	if err == ErrNotFound {
		return r.db.Insert(FeatureFlagType, ff)
	}
	if err != nil {
		return err
	}
	return ErrAlreadyExists
}

func (r *FeatureFlagRepository) Get(name FeatureFlagName) (*FeatureFlag, error) {
	raw, err := r.db.First(FeatureFlagType, PK, name)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw.(*FeatureFlag), nil
}

func (r *FeatureFlagRepository) Delete(name FeatureFlagName) error {
	// TODO Cannot be deleted when in use by role, tenant, or project
	featureFlag, err := r.Get(name)
	if err != nil {
		return err
	}
	return r.db.Delete(FeatureFlagType, featureFlag)
}

func (r *FeatureFlagRepository) save(ff *FeatureFlag) error {
	return r.db.Insert(FeatureFlagType, ff)
}

func (r *FeatureFlagRepository) delete(name string) error {
	featureFlag, err := r.Get(name)
	if err != nil {
		return err
	}
	return r.db.Delete(FeatureFlagType, featureFlag)
}

func (r *FeatureFlagRepository) List() ([]FeatureFlagName, error) {
	iter, err := r.db.Get(FeatureFlagType, PK)
	if err != nil {
		return nil, err
	}

	list := []FeatureFlagName{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		ff := raw.(*FeatureFlag)
		list = append(list, ff.Name)
	}
	return list, nil
}

func (r *FeatureFlagRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.delete(objID)
	}

	ff := &FeatureFlag{}
	err := json.Unmarshal(data, ff)
	if err != nil {
		return err
	}

	return r.save(ff)
}
