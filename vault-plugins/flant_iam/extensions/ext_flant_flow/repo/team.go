package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	TeamForeignPK = "team_uuid"
)

func TeamSchema() map[string]*hcmemdb.TableSchema {
	return map[string]*hcmemdb.TableSchema{
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

func (r *TeamRepository) Delete(id model.TeamUUID, archivingTimestamp model.UnixTime, archivingHash int64) error {
	team, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if team.IsDeleted() {
		return consts.ErrIsArchived
	}
	team.ArchivingTimestamp = archivingTimestamp
	team.ArchivingHash = archivingHash
	return r.Update(team)
}

func (r *TeamRepository) List(showArchived bool) ([]*model.Team, error) {
	iter, err := r.db.Get(model.TeamType, PK)
	if err != nil {
		return nil, err
	}

	list := []*model.Team{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Team)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *TeamRepository) ListIDs(showArchived bool) ([]model.TeamUUID, error) {
	objs, err := r.List(showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.TeamUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *TeamRepository) Iter(action func(*model.Team) (bool, error)) error {
	iter, err := r.db.Get(model.TeamType, PK)
	if err != nil {
		return err
	}

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
	if team.ArchivingTimestamp == 0 {
		return nil, consts.ErrIsNotArchived
	}
	team.ArchivingTimestamp = 0
	team.ArchivingHash = 0
	err = r.Update(team)
	if err != nil {
		return nil, err
	}
	return team, nil
}
