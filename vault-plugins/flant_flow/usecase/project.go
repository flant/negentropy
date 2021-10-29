package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_flow/iam_client"
	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ProjectService struct {
	db            *io.MemoryStoreTxn
	projectClient iam_client.Projects
}

func Projects(db *io.MemoryStoreTxn, projectClient iam_client.Projects) *ProjectService {
	return &ProjectService{db: db, projectClient: projectClient}
}

func (s *ProjectService) Create(project *model.Project) error {
	// Verify client(tenant) exists
	_, err := repo.NewClientRepository(s.db).GetByID(project.TenantUUID)
	if err != nil {
		return err
	}

	project.Version = repo.NewResourceVersion()
	// TODO verify servicepacks
	// TODO fix servicepacks params due to default teams

	return repo.NewProjectRepository(s.db).Create(project)
}

func (s *ProjectService) Update(project *model.Project) error {
	repository := repo.NewProjectRepository(s.db)

	stored, err := repository.GetByID(project.UUID)
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
	project.Version = repo.NewResourceVersion()
	// TODO verify servicepacks
	// Update

	return repository.Update(project)
}

func (s *ProjectService) Delete(id model.ProjectUUID) error {
	archivingTimestamp, archivingHash := ArchivingLabel()
	return repo.NewProjectRepository(s.db).Delete(id, archivingTimestamp, archivingHash)
}

func (s *ProjectService) List(cid model.ClientUUID, showArchived bool) ([]*model.Project, error) {
	return repo.NewProjectRepository(s.db).List(cid, showArchived)
}

func (s *ProjectService) GetByID(pid model.ProjectUUID) (*model.Project, error) {
	return repo.NewProjectRepository(s.db).GetByID(pid)
}

func (s *ProjectService) Restore(id model.ProjectUUID) (*model.Project, error) {
	return repo.NewProjectRepository(s.db).Restore(id)
}
