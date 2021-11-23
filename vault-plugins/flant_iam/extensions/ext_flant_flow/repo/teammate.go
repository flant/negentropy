package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func TeammateSchema() map[string]*hcmemdb.TableSchema {
	return map[string]*hcmemdb.TableSchema{
		model.TeammateType: {
			Name: model.TeammateType,
			Indexes: map[string]*hcmemdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &hcmemdb.UUIDFieldIndex{
						Field: "UUID",
					},
				},
				"identifier": {
					Name:   "identifier",
					Unique: true,
					Indexer: &hcmemdb.StringFieldIndex{
						Field:     "Identifier",
						Lowercase: true,
					},
				},
				"version": {
					Name: "version",
					Indexer: &hcmemdb.StringFieldIndex{
						Field: "Version",
					},
				},
				TeamForeignPK: {
					Name: TeamForeignPK,
					Indexer: &hcmemdb.StringFieldIndex{
						Field: "TeamUUID",
					},
				},
			},
		},
	}
}

type TeammateRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewTeammateRepository(tx *io.MemoryStoreTxn) *TeammateRepository {
	return &TeammateRepository{db: tx}
}

func (r *TeammateRepository) save(teammate *model.Teammate) error {
	return r.db.Insert(model.TeammateType, teammate)
}

func (r *TeammateRepository) Create(teammate *model.Teammate) error {
	return r.save(teammate)
}

func (r *TeammateRepository) GetRawByID(id iam_model.UserUUID) (interface{}, error) {
	raw, err := r.db.First(model.TeammateType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *TeammateRepository) GetByID(id iam_model.UserUUID) (*model.Teammate, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Teammate), err
}

func (r *TeammateRepository) Update(teammate *model.Teammate) error {
	_, err := r.GetByID(teammate.UUID)
	if err != nil {
		return err
	}
	return r.save(teammate)
}

func (r *TeammateRepository) Delete(id iam_model.UserUUID, archiveMark memdb.ArchiveMark) error {
	teammate, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if teammate.IsDeleted() {
		return consts.ErrIsArchived
	}
	teammate.Timestamp = archiveMark.Timestamp
	teammate.Hash = archiveMark.Hash
	return r.Update(teammate)
}

func (r *TeammateRepository) List(showArchived bool) ([]*model.Teammate, error) {
	iter, err := r.db.Get(model.TeammateType, PK)
	if err != nil {
		return nil, err
	}

	list := []*model.Teammate{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Teammate)
		if showArchived || obj.Timestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *TeammateRepository) ListIDs(showArchived bool) ([]iam_model.UserUUID, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]iam_model.UserUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *TeammateRepository) Iter(action func(*model.Teammate) (bool, error)) error {
	iter, err := r.db.Get(model.TeammateType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Teammate)
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

func (r *TeammateRepository) Sync(_ string, data []byte) error {
	teammate := &model.Teammate{}
	err := json.Unmarshal(data, teammate)
	if err != nil {
		return err
	}

	return r.save(teammate)
}

func (r *TeammateRepository) Restore(id iam_model.UserUUID) (*model.Teammate, error) {
	teammate, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if teammate.Timestamp == 0 {
		return nil, consts.ErrIsNotArchived
	}
	teammate.Timestamp = 0
	teammate.Hash = 0
	err = r.Update(teammate)
	if err != nil {
		return nil, err
	}
	return teammate, nil
}
