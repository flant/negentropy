package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

// TODO add work with IAM

type TeamService struct {
	repo *repo.TeamRepository

	// subteams
	childrenDeleters []DeleterByParent
}

func Teams(db *io.MemoryStoreTxn) *TeamService {
	return &TeamService{
		repo:             repo.NewTeamRepository(db),
		childrenDeleters: []DeleterByParent{},
	}
}

func (s *TeamService) Create(t *model.Team) error {
	t.Version = repo.NewResourceVersion()
	return s.repo.Create(t)
}

func (s *TeamService) Update(updated *model.Team) error {
	stored, err := s.repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}

	// Validate

	if stored.Version != updated.Version {
		return model.ErrBadVersion
	}
	updated.Version = repo.NewResourceVersion()

	// Update

	return s.repo.Create(updated)
}

func (s *TeamService) Delete(id model.TeamUUID) error {
	// TODO:
	// Check no child
	// Check no teammates
	// Check not default for any feature_flag
	// Delete all child IAM.group & IAM.rolebinding

	archivingTimestamp, archivingHash := ArchivingLabel()
	if err := deleteChildren(id, s.childrenDeleters, archivingTimestamp, archivingHash); err != nil {
		return err
	}
	return s.repo.Delete(id, archivingTimestamp, archivingHash)
}

func (s *TeamService) GetByID(id model.TeamUUID) (*model.Team, error) {
	return s.repo.GetByID(id)
}

func (s *TeamService) List(showArchived bool) ([]*model.Team, error) {
	return s.repo.List(showArchived)
}

func (s *TeamService) Restore(id model.TeamUUID, fullRestore bool) (*model.Team, error) {
	if fullRestore {
		// TODO check if full restore available
		// TODO FullRestore
		return s.repo.Restore(id)
	}
	// TODO Short Restore
	return s.repo.Restore(id)
}
