package model

import (
	"encoding/json"

	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	ProjectType = "project" // also, memdb schema name

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
}

func (p *Project) ObjType() string {
	return ProjectType
}

func (p *Project) ObjId() string {
	return p.UUID
}

type ProjectRepository struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	tenantRepo *TenantRepository
}

func NewProjectRepository(tx *io.MemoryStoreTxn) *ProjectRepository {
	return &ProjectRepository{
		db:         tx,
		tenantRepo: NewTenantRepository(tx),
	}
}

func (r *ProjectRepository) save(project *Project) error {
	return r.db.Insert(ProjectType, project)
}

func (r *ProjectRepository) GetByID(id ProjectUUID) (*Project, error) {
	raw, err := r.db.First(ProjectType, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	project := raw.(*Project)

	return project, nil
}

func (r *ProjectRepository) Create(project *Project) error {
	return r.save(project)
}

func (r *ProjectRepository) Update(project *Project) error {
	_, err := r.GetByID(project.UUID)
	if err != nil {
		return err
	}

	return r.save(project)
}

func (r *ProjectRepository) Delete(id string) error {
	project, err := r.GetByID(id)
	if err != nil {
		return err
	}

	return r.db.Delete(ProjectType, project)
}

func (r *ProjectRepository) List(tenantID TenantUUID) ([]*Project, error) {
	iter, err := r.db.Get(ProjectType, TenantForeignPK, tenantID)
	if err != nil {
		return nil, err
	}

	list := []*Project{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		project := raw.(*Project)
		list = append(list, project)
	}
	return list, nil
}

func (r *ProjectRepository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	pr := &Project{}
	err := json.Unmarshal(data, pr)
	if err != nil {
		return err
	}

	return r.save(pr)
}
