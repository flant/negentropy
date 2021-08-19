package repo

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func FeatureFlagSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			model.FeatureFlagType: {
				Name: model.FeatureFlagType,
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

type FeatureFlagRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewFeatureFlagRepository(tx *io.MemoryStoreTxn) *FeatureFlagRepository {
	return &FeatureFlagRepository{db: tx}
}

func (r *FeatureFlagRepository) save(ff *model.FeatureFlag) error {
	return r.db.Insert(model.FeatureFlagType, ff)
}

func (r *FeatureFlagRepository) Create(ff *model.FeatureFlag) error {
	return r.save(ff)
}

func (r *FeatureFlagRepository) GetRawByID(id model.FeatureFlagName) (interface{}, error) {
	raw, err := r.db.First(model.FeatureFlagType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *FeatureFlagRepository) GetByID(id model.FeatureFlagName) (*model.FeatureFlag, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.FeatureFlag), err
}

func (r *FeatureFlagRepository) Update(ff *model.FeatureFlag) error {
	_, err := r.GetByID(ff.Name)
	if err != nil {
		return err
	}
	return r.save(ff)
}

func (r *FeatureFlagRepository) Delete(id model.FeatureFlagName, archivingTimestamp model.UnixTime, archivingHash int64) error {
	ff, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if ff.IsDeleted() {
		return model.ErrIsArchived
	}
	ff.ArchivingTimestamp = archivingTimestamp
	ff.ArchivingHash = archivingHash
	return r.Update(ff)
}

func (r *FeatureFlagRepository) List(showArchived bool) ([]*model.FeatureFlag, error) {
	iter, err := r.db.Get(model.FeatureFlagType, PK)
	if err != nil {
		return nil, err
	}

	list := []*model.FeatureFlag{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.FeatureFlag)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *FeatureFlagRepository) ListIDs(showArchived bool) ([]model.FeatureFlagName, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.FeatureFlagName, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *FeatureFlagRepository) Iter(action func(*model.FeatureFlag) (bool, error)) error {
	iter, err := r.db.Get(model.FeatureFlagType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.FeatureFlag)
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
	ff := &model.FeatureFlag{}
	err := json.Unmarshal(data, ff)
	if err != nil {
		return err
	}

	return r.save(ff)
}
