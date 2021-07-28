package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	ProjectForeignPK = "project_uuid"
)

func ProjectSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			ProjectType: {
				Name: ProjectType,
				Indexes: map[string]*memdb.IndexSchema{
					PK: {
						Name:   PK,
						Unique: true,
						Indexer: &memdb.UUIDFieldIndex{
							Field: "UUID",
						},
					},
					TenantForeignPK: {
						Name: TenantForeignPK,
						Indexer: &memdb.StringFieldIndex{
							Field:     "TenantUUID",
							Lowercase: true,
						},
					},
					"version": {
						Name: "version",
						Indexer: &memdb.StringFieldIndex{
							Field: "Version",
						},
					},
					"identifier": {
						Name: "identifier",
						Indexer: &memdb.StringFieldIndex{
							Field: "Identifier",
						},
					},
				},
			},
		},
	}
}

type Project struct {
	UUID       ProjectUUID `json:"uuid"` // PK
	TenantUUID TenantUUID  `json:"tenant_uuid"`
	Version    string      `json:"resource_version"`
	Identifier string      `json:"identifier"`

	FeatureFlags []FeatureFlag `json:"feature_flags"`

	ArchivingTimestamp UnixTime `json:"archiving_timestamp"`
	ArchivingHash      int64    `json:"archiving_hash"`
}

const ProjectType = "project" // also, memdb schema name

func (u *Project) ObjType() string {
	return ProjectType
}

func (u *Project) ObjId() string {
	return u.UUID
}

type ProjectRepository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func NewProjectRepository(tx *io.MemoryStoreTxn) *ProjectRepository {
	return &ProjectRepository{db: tx}
}

func (r *ProjectRepository) save(project *Project) error {
	return r.db.Insert(ProjectType, project)
}

func (r *ProjectRepository) Create(project *Project) error {
	return r.save(project)
}

func (r *ProjectRepository) GetRawByID(id ProjectUUID) (interface{}, error) {
	raw, err := r.db.First(ProjectType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *ProjectRepository) GetByID(id ProjectUUID) (*Project, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*Project), err
}

func (r *ProjectRepository) Update(project *Project) error {
	_, err := r.GetByID(project.UUID)
	if err != nil {
		return err
	}
	return r.save(project)
}

func (r *ProjectRepository) Delete(id ProjectUUID, archivingTimestamp UnixTime, archivingHash int64) error {
	project, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if project.ArchivingTimestamp != 0 {
		return ErrIsArchived
	}
	project.ArchivingTimestamp = archivingTimestamp
	project.ArchivingHash = archivingHash
	return r.Update(project)
}

func (r *ProjectRepository) List(tenantUUID TenantUUID, showArchived bool) ([]*Project, error) {
	iter, err := r.db.Get(ProjectType, TenantForeignPK, tenantUUID)
	if err != nil {
		return nil, err
	}

	list := []*Project{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Project)
		if showArchived || obj.ArchivingTimestamp == 0 {
			list = append(list, obj)
		}
	}
	return list, nil
}

func (r *ProjectRepository) ListIDs(tenantID TenantUUID, showArchived bool) ([]ProjectUUID, error) {
	objs, err := r.List(tenantID, showArchived)
	if err != nil {
		return nil, err
	}
	ids := make([]ProjectUUID, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *ProjectRepository) Iter(action func(*Project) (bool, error)) error {
	iter, err := r.db.Get(ProjectType, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*Project)
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
	project := &Project{}
	err := json.Unmarshal(data, project)
	if err != nil {
		return err
	}

	return r.save(project)
}

func (r *ProjectRepository) Restore(id ProjectUUID) (*Project, error) {
	project, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}
	if project.ArchivingTimestamp == 0 {
		return nil, ErrIsNotArchived
	}
	project.ArchivingTimestamp = 0
	project.ArchivingHash = 0
	err = r.Update(project)
	if err != nil {
		return nil, err
	}
	return project, nil
}
