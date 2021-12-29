package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	model2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func MultipassGenerationNumberSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model2.MultipassGenerationNumberType: {
				Name: model2.MultipassGenerationNumberType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					"generation_number": {
						Name:   "generation_number",
						Unique: true,
						Indexer: &hcmemdb.IntFieldIndex{
							Field: "GenerationNumber",
						},
					},
				},
			},
		},
	}
}

type MultipassGenerationNumberRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewMultipassGenerationNumberRepository(tx *io.MemoryStoreTxn) *MultipassGenerationNumberRepository {
	return &MultipassGenerationNumberRepository{
		db: tx,
	}
}

func (r *MultipassGenerationNumberRepository) save(t *model2.MultipassGenerationNumber) error {
	return r.db.Insert(model2.MultipassGenerationNumberType, t)
}

func (r *MultipassGenerationNumberRepository) Create(t *model2.MultipassGenerationNumber) error {
	return r.db.Insert(model2.MultipassGenerationNumberType, t)
}

func (r *MultipassGenerationNumberRepository) GetByID(id model2.MultipassGenerationNumberUUID) (*model2.MultipassGenerationNumber, error) {
	raw, err := r.db.First(model2.MultipassGenerationNumberType, ID, id)
	if err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, nil
	}

	return raw.(*model2.MultipassGenerationNumber), nil
}

func (r *MultipassGenerationNumberRepository) Update(updated *model2.MultipassGenerationNumber) error {
	raw, err := r.GetByID(updated.UUID)
	if err != nil {
		return err
	}

	if raw == nil {
		return consts.ErrNotFound
	}

	// Update

	return r.db.Insert(model2.MultipassGenerationNumberType, updated)
}

func (r *MultipassGenerationNumberRepository) Delete(id model2.MultipassGenerationNumberUUID) error {
	return r.delete(id)
}

func (r *MultipassGenerationNumberRepository) delete(id string) error {
	mp, err := r.GetByID(id)
	if err != nil {
		return err
	}

	if mp == nil {
		return consts.ErrNotFound
	}

	return r.db.Delete(model2.MultipassGenerationNumberType, mp)
}

func (r *MultipassGenerationNumberRepository) List() ([]*model2.MultipassGenerationNumber, error) {
	iter, err := r.db.Get(model2.MultipassGenerationNumberType, ID)
	if err != nil {
		return nil, err
	}

	list := make([]*model2.MultipassGenerationNumber, 0)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model2.MultipassGenerationNumber)
		list = append(list, t)
	}
	return list, nil
}

func (r *MultipassGenerationNumberRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.delete(objID)
	}

	obj := &model2.MultipassGenerationNumber{}
	err := json.Unmarshal(data, obj)
	if err != nil {
		return err
	}

	return r.save(obj)
}
