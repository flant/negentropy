package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	MultipassGenerationNumberType = "tokengenerationnumber" // also, memdb schema name
)

func MultipassGenerationNumberSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			MultipassGenerationNumberType: {
				Name: MultipassGenerationNumberType,
				Indexes: map[string]*memdb.IndexSchema{
					ID: {
						Name:   ID,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					"generation_number": {
						Name:   "generation_number",
						Unique: true,
						Indexer: &memdb.IntFieldIndex{
							Field: "GenerationNumber",
						},
					},
				},
			},
		},
	}
}

// MultipassGenerationNumber
// This entity is a 1-to-1 relation with Multipass.
type MultipassGenerationNumber struct {
	UUID             MultipassGenerationNumberUUID `json:"uuid"` // PK == multipass uuid.
	GenerationNumber int                           `json:"generation_number"`
}

func (t *MultipassGenerationNumber) ObjType() string {
	return MultipassGenerationNumberType
}

func (t *MultipassGenerationNumber) ObjId() string {
	return t.UUID
}

type MultipassGenerationNumberRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewMultipassGenerationNumberRepository(tx *io.MemoryStoreTxn) *MultipassGenerationNumberRepository {
	return &MultipassGenerationNumberRepository{
		db: tx,
	}
}

func (r *MultipassGenerationNumberRepository) save(t *MultipassGenerationNumber) error {
	return r.db.Insert(MultipassGenerationNumberType, t)
}

func (r *MultipassGenerationNumberRepository) Create(t *MultipassGenerationNumber) error {
	return r.db.Insert(MultipassGenerationNumberType, t)
}

func (r *MultipassGenerationNumberRepository) GetByID(id MultipassGenerationNumberUUID) (*MultipassGenerationNumber, error) {
	raw, err := r.db.First(MultipassGenerationNumberType, ID, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw.(*MultipassGenerationNumber), nil
}

func (r *MultipassGenerationNumberRepository) Update(updated *MultipassGenerationNumber) error {
	_, err := r.GetByID(updated.UUID)
	if err != nil {
		return err
	}

	// Update

	return r.db.Insert(MultipassGenerationNumberType, updated)
}

func (r *MultipassGenerationNumberRepository) Delete(id MultipassGenerationNumberUUID) error {
	return r.delete(id)
}

func (r *MultipassGenerationNumberRepository) delete(id string) error {
	tenant, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(MultipassGenerationNumberType, tenant)
}

func (r *MultipassGenerationNumberRepository) List() ([]*MultipassGenerationNumber, error) {
	iter, err := r.db.Get(MultipassGenerationNumberType, ID)
	if err != nil {
		return nil, err
	}

	list := make([]*MultipassGenerationNumber, 0)
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*MultipassGenerationNumber)
		list = append(list, t)
	}
	return list, nil
}

func (r *MultipassGenerationNumberRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.delete(objID)
	}

	obj := &MultipassGenerationNumber{}
	err := json.Unmarshal(data, obj)
	if err != nil {
		return err
	}

	return r.save(obj)
}
