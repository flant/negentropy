package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type ProjectService struct {
	db     *io.MemoryStoreTxn
	origin consts.ObjectOrigin
}

func Projects(db *io.MemoryStoreTxn, origin consts.ObjectOrigin) *ProjectService {
	return &ProjectService{db: db, origin: origin}
}

func (s *ProjectService) Create(project *model.Project) error {
	// Verify tenant exists
	_, err := repo.NewTenantRepository(s.db).GetByID(project.TenantUUID)
	if err != nil {
		return err
	}

	project.Version = repo.NewResourceVersion()
	project.Origin = s.origin

	return repo.NewProjectRepository(s.db).Create(project)
}

func (s *ProjectService) Update(project *model.Project) error {
	repository := repo.NewProjectRepository(s.db)

	stored, err := repository.GetByID(project.UUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	// Validate
	if stored.TenantUUID != project.TenantUUID {
		return consts.ErrNotFound
	}
	if stored.Version != project.Version {
		return consts.ErrBadVersion
	}
	if stored.Origin != s.origin {
		return consts.ErrBadOrigin
	}
	project.Version = repo.NewResourceVersion()

	// Update

	return repository.Update(project)
}

func (s *ProjectService) Delete(id model.ProjectUUID) error {
	repository := repo.NewProjectRepository(s.db)
	stored, err := repository.GetByID(id)
	if err != nil {
		return err
	}
	if stored.Origin != s.origin {
		return consts.ErrBadOrigin
	}

	return repo.NewProjectRepository(s.db).Delete(id, memdb.NewArchiveMark())
}

func (s *ProjectService) List(tid model.TenantUUID, showArchived bool) ([]*model.Project, error) {
	return repo.NewProjectRepository(s.db).List(tid, showArchived)
}

func (s *ProjectService) GetByID(pid model.ProjectUUID) (*model.Project, error) {
	return repo.NewProjectRepository(s.db).GetByID(pid)
}

func (s *ProjectService) Restore(id model.ProjectUUID) (*model.Project, error) {
	repository := repo.NewProjectRepository(s.db)
	stored, err := repository.GetByID(id)
	if err != nil {
		return nil, err
	}
	if stored.Origin != s.origin {
		return nil, consts.ErrBadOrigin
	}
	return repo.NewProjectRepository(s.db).Restore(id)
}
