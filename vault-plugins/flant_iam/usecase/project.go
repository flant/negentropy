package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ProjectService struct {
	db *io.MemoryStoreTxn
}

func Projects(db *io.MemoryStoreTxn) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) Create(project *model.Project) error {
	// Verify tenant exists
	_, err := model.NewTenantRepository(s.db).GetByID(project.TenantUUID)
	if err != nil {
		return err
	}

	project.Version = model.NewResourceVersion()

	return model.NewProjectRepository(s.db).Create(project)
}

func (s *ProjectService) Update(project *model.Project) error {
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

func (s *ProjectService) Delete(id model.ProjectUUID) error {
	archivingTimestamp, archivingHash := ArchivingLabel()
	return model.NewProjectRepository(s.db).Delete(id, archivingTimestamp, archivingHash)
}

func (s *ProjectService) List(tid model.TenantUUID, showArchived bool) ([]*model.Project, error) {
	return model.NewProjectRepository(s.db).List(tid, showArchived)
}

func (s *ProjectService) GetByID(pid model.ProjectUUID) (*model.Project, error) {
	return model.NewProjectRepository(s.db).GetByID(pid)
}

func (s *ProjectService) Restore(id model.ProjectUUID) (*model.Project, error) {
	return model.NewProjectRepository(s.db).Restore(id)
}
