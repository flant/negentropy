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

func TeammateSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.TeammateType: {
				Name: model.TeammateType,
				Indexes: map[string]*hcmemdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "UserUUID",
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
						Indexer: &hcmemdb.UUIDFieldIndex{
							Field: "TeamUUID",
						},
					},
				},
			},
		},
		MandatoryForeignKeys: map[string][]memdb.Relation{
			model.TeammateType: {
				{
					OriginalDataTypeFieldName:     "UserUUID",
					RelatedDataType:               iam_model.UserType,
					RelatedDataTypeFieldIndexName: PK,
				},
				{
					OriginalDataTypeFieldName:     "TeamUUID",
					RelatedDataType:               model.TeamType,
					RelatedDataTypeFieldIndexName: PK,
				},
			},
		},
		CascadeDeletes: map[string][]memdb.Relation{
			model.TeammateType: {
				{
					OriginalDataTypeFieldName:     "UserUUID",
					RelatedDataType:               iam_model.UserType,
					RelatedDataTypeFieldIndexName: PK,
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
	stored, err := r.GetByID(teammate.UserUUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	return r.save(teammate)
}

func (r *TeammateRepository) Delete(id iam_model.UserUUID, archiveMark memdb.ArchiveMark) error {
	teammate, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if teammate.Archived() {
		return consts.ErrIsArchived
	}

	return r.db.Archive(model.TeammateType, teammate, archiveMark)
}

func (r *TeammateRepository) List(teamID model.TeamUUID, showArchived bool) ([]*model.Teammate, error) {
	iter, err := r.db.Get(model.TeammateType, TeamForeignPK, teamID)
	if err != nil {
		return nil, err
	}

	list := []*model.Teammate{}
	err = r.Iter(iter, func(teammate *model.Teammate) (bool, error) {
		if showArchived || teammate.NotArchived() {
			list = append(list, teammate)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (r *TeammateRepository) ListIDs(showArchived bool) ([]iam_model.UserUUID, error) {
	iter, err := r.db.Get(model.TeammateType, PK)
	if err != nil {
		return nil, err
	}
	ids := []iam_model.UserUUID{}
	err = r.Iter(iter, func(teammate *model.Teammate) (bool, error) {
		if showArchived || teammate.NotArchived() {
			ids = append(ids, teammate.UserUUID)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *TeammateRepository) Iter(iter hcmemdb.ResultIterator, action func(*model.Teammate) (bool, error)) error {
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
	if teammate.NotArchived() {
		return nil, consts.ErrIsNotArchived
	}
	err = r.db.CascadeRestore(model.TeammateType, teammate)
	if err != nil {
		return nil, err
	}
	return teammate, nil
}

func (r *TeammateRepository) ListAll(showArchived bool) ([]*model.Teammate, error) {
	iter, err := r.db.Get(model.TeammateType, PK)
	if err != nil {
		return nil, err
	}
	list := []*model.Teammate{}
	err = r.Iter(iter, func(teammate *model.Teammate) (bool, error) {
		if showArchived || teammate.NotArchived() {
			list = append(list, teammate)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}
