package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

const (
	TeamForeignPK   = "team_uuid"
	ParentTeamIndex = "parent_team_uuid"
)

func TeamSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*hcmemdb.TableSchema{
			model.TeamType: {
				Name: model.TeamType,
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
					ParentTeamIndex: {
						Name:         ParentTeamIndex,
						AllowMissing: true,
						Indexer: &hcmemdb.StringFieldIndex{
							Field: "ParentTeamUUID",
						},
					},
				},
			},
		},
		CheckingRelations: map[string][]memdb.Relation{
			model.TeamType: {
				{
					OriginalDataTypeFieldName:     "UUID",
					RelatedDataType:               model.TeammateType,
					RelatedDataTypeFieldIndexName: TeamForeignPK,
				},
				{
					OriginalDataTypeFieldName:     "ParentTeamUUID",
					RelatedDataType:               model.TeamType,
					RelatedDataTypeFieldIndexName: ParentTeamIndex,
				},
			},
		},
	}
}

type TeamRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewTeamRepository(tx *io.MemoryStoreTxn) *TeamRepository {
	return &TeamRepository{db: tx}
}

func (r *TeamRepository) save(team *model.Team) error {
	return r.db.Insert(model.TeamType, team)
}

func (r *TeamRepository) Create(team *model.Team) error {
	return r.save(team)
}

func (r *TeamRepository) GetRawByID(id model.TeamUUID) (interface{}, error) {
	raw, err := r.db.First(model.TeamType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, consts.ErrNotFound
	}
	return raw, nil
}

func (r *TeamRepository) GetByID(id model.TeamUUID) (*model.Team, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Team), err
}

func (r *TeamRepository) Update(team *model.Team) error {
	_, err := r.GetByID(team.UUID)
	if err != nil {
		return err
	}
	return r.save(team)
}

func (r *TeamRepository) Delete(id model.TeamUUID, archiveMark memdb.ArchiveMark) error {
	team, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if team.Archived() {
		return consts.ErrIsArchived
	}
	return r.db.Archive(model.TeamType, team, archiveMark)
}

func (r *TeamRepository) List(showArchived bool) ([]*model.Team, error) {
	iter, err := r.db.Get(model.TeamType, PK)
	if err != nil {
		return nil, err
	}

	list := []*model.Team{}
	err = r.Iter(iter, func(team *model.Team) (bool, error) {
		if showArchived || !team.Archived() {
			list = append(list, team)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (r *TeamRepository) ListIDs(showArchived bool) ([]model.TeamUUID, error) {
	iter, err := r.db.Get(model.TeamType, PK)
	if err != nil {
		return nil, err
	}
	ids := []model.TeamUUID{}
	err = r.Iter(iter, func(team *model.Team) (bool, error) {
		if showArchived || !team.Archived() {
			ids = append(ids, team.UUID)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// Iter operate action under each of iter.next() entity
func (r *TeamRepository) Iter(iter hcmemdb.ResultIterator, action func(*model.Team) (bool, error)) error {
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Team)
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

func (r *TeamRepository) Sync(_ string, data []byte) error {
	team := &model.Team{}
	err := json.Unmarshal(data, team)
	if err != nil {
		return err
	}

	return r.save(team)
}

func (r *TeamRepository) Restore(id model.TeamUUID) (*model.Team, error) {
	team, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !team.Archived() {
		return nil, consts.ErrIsNotArchived
	}
	err = r.db.Restore(model.TeamType, team)
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (r *TeamRepository) ListChildTeamIDs(parentTeamUUID model.TeamUUID, showArchived bool) ([]model.TeamUUID, error) {
	iter, err := r.db.Get(model.TeamType, ParentTeamIndex, parentTeamUUID)
	if err != nil {
		return nil, err
	}
	ids := []model.TeamUUID{}
	err = r.Iter(iter, func(team *model.Team) (bool, error) {
		if showArchived || !team.Archived() {
			ids = append(ids, team.UUID)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}
