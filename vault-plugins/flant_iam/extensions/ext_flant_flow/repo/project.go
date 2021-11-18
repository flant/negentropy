package repo

import (
	"encoding/json"

	hcmemdb "github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const ProjectForeignPK = "project_uuid"

func ProjectSchema() map[string]*hcmemdb.TableSchema {
	return map[string]*hcmemdb.TableSchema{
		model.ProjectType: {
			Name: model.ProjectType,
			Indexes: map[string]*hcmemdb.IndexSchema{
				PK: {
					Name:   PK,
					Unique: true,
					Indexer: &hcmemdb.UUIDFieldIndex{
						Field: "UUID",
					},
				},
				ClientForeignPK: {
					Name: ClientForeignPK,
					Indexer: &hcmemdb.StringFieldIndex{
						Field:     "TenantUUID",
						Lowercase: true,
					},
				},
				"version": {
					Name: "version",
					Indexer: &hcmemdb.StringFieldIndex{
						Field: "Version",
					},
				},
				"identifier": {
					Name: "identifier",
					Indexer: &hcmemdb.StringFieldIndex{
						Field: "Identifier",
					},
				},
			},
		},
	}
}

type ProjectRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewProjectRepository(tx *io.MemoryStoreTxn) *ProjectRepository {
	return &ProjectRepository{db: tx}
}

func (r *ProjectRepository) save(project *model.Project) error {
	return r.db.Insert(model.ProjectType, project)
}

func (r *ProjectRepository) Create(project *model.Project) error {
	return r.save(project)
}

func (r *ProjectRepository) GetRawByID(id model.ProjectUUID) (interface{}, error) {
	raw, err := r.db.First(model.ProjectType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, model.ErrNotFound
	}
	return raw, nil
}

func (r *ProjectRepository) GetByID(id model.ProjectUUID) (*model.Project, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*model.Project), err
}

func (r *ProjectRepository) Update(project *model.Project) error {
	_, err := r.GetByID(project.UUID)
	if err != nil {
		return err
	}
	return r.save(project)
}

func (r *ProjectRepository) Delete(id model.ProjectUUID, archivingTimestamp model.UnixTime, archivingHash int64) error {
	project, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if project.IsDeleted() {
		return model.ErrIsArchived
	}
	project.ArchivingTimestamp = archivingTimestamp
	project.ArchivingHash = archivingHash
	return r.Update(project)
}

func (r *ProjectRepository) List(clientUUID model.ClientUUID, showArchived bool) ([]*model.Project, error) {
	iter, err := r.db.Get(model.ProjectType, ClientForeignPK, clientUUID)
	if err != nil {
		return nil, err
	}

	list := []*model.Project{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Project)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *ProjectRepository) ListIDs(clientUUID model.ClientUUID, showArchived bool) ([]model.ProjectUUID, error) {
	objs, err := r.List(clientUUID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]model.ProjectUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ProjectRepository) Iter(action func(*model.Project) (bool, error)) error {
	iter, err := r.db.Get(model.ProjectType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*model.Project)
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

func (r *ProjectRepository) Sync(objID string, data []byte) error {
	project := &model.Project{}
	err := json.Unmarshal(data, project)
	if err != nil {
		return err
	}

	return r.save(project)
}

func (r *ProjectRepository) Restore(id model.ProjectUUID) (*model.Project, error) {
	project, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if project.ArchivingTimestamp == 0 {
		return nil, model.ErrIsNotArchived
	}
	project.ArchivingTimestamp = 0
	project.ArchivingHash = 0
	err = r.Update(project)
	if err != nil {
		return nil, err
	}
	return project, nil
}
