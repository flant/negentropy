package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ProjectsService struct {
	db *io.MemoryStoreTxn
}

func Projects(db *io.MemoryStoreTxn) *ProjectsService {
	return &ProjectsService{db: db}
}

func (s *ProjectsService) Create(project *model.Project) error {
	// Verify tenant exists
	_, err := model.NewTenantRepository(s.db).GetByID(project.TenantUUID)
	if err != nil {
		return err
	}

	project.Version = model.NewResourceVersion()

	return model.NewProjectRepository(s.db).Create(project)
}

func (s *ProjectsService) Update(project *model.Project) error {
	repo := model.NewProjectRepository(s.db)

	stored, err := repo.GetByID(project.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != project.TenantUUID {
		return model.ErrNotFound
	}
	if stored.Version != project.Version {
		return model.ErrBadVersion
	}
	project.Version = model.NewResourceVersion()

	// Update

	return repo.Update(project)
}

func (s *ProjectsService) Delete(id string) error {
	return model.NewProjectRepository(s.db).Delete(id)
}

func (s *ProjectsService) List(tid model.TenantUUID) ([]*model.Project, error) {
	return model.NewProjectRepository(s.db).List(tid)
}

func (s *ProjectsService) GetByID(pid model.ProjectUUID) (*model.Project, error) {
	return model.NewProjectRepository(s.db).GetByID(pid)
}
