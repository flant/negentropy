// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type Project-parentType Tenant
// When: 2021-07-16 14:19:25.123137 +0300 MSK m=+0.000177046
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ProjectUUID = string 

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

func (r *ProjectRepository) Delete(id ProjectUUID) error {
	project, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete(ProjectType, project)
}

func (r *ProjectRepository) List(tenantUUID TenantUUID) ([]*Project, error) {
	
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
		list = append(list, obj)
	}
	return list, nil
}

func (r *ProjectRepository) ListIDs(tenantID TenantUUID) ([]ProjectUUID, error) {
	objs, err := r.List(tenantID)
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
	if data == nil {
		return r.Delete(objID)
	}

	project := &Project{}
	err := json.Unmarshal(data, project)
	if err != nil {
		return err
	}

	return r.save(project)
}