package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type TeamService struct {
	repo *repo.TeamRepository
}

func Teams(db *io.MemoryStoreTxn) *TeamService {
	return &TeamService{
		repo: repo.NewTeamRepository(db),
	}
}

func (s *TeamService) Create(t *model.Team) error {
	err := s.validateParentTeamUUID(t)
	if err != nil {
		return err
	}
	t.Version = repo.NewResourceVersion()
	return s.repo.Create(t)
}

func (s *TeamService) validateParentTeamUUID(t *model.Team) error {
	if t.ParentTeamUUID != "" {
		_, err := s.repo.GetByID(t.ParentTeamUUID)
		if errors.Is(err, consts.ErrNotFound) {
			return fmt.Errorf("%w: parent_team_uuid must be valid team uuid or empty", consts.ErrWrongArg)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *TeamService) Update(updated *model.Team) error {
	err := s.validateParentTeamUUID(updated)
	if err != nil {
		return err
	}
	stored, err := s.repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}
	updated.TeamType = stored.TeamType // team_type cant't be changed
	if stored.Version != updated.Version {
		return consts.ErrBadVersion
	}
	updated.Version = repo.NewResourceVersion()
	return s.repo.Create(updated)
}

func (s *TeamService) Delete(id model.TeamUUID) error {
	team, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if team.Archived() {
		return consts.ErrIsArchived
	}

	// Check no child
	children, err := s.repo.ListChildTeamIDs(id, false)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		return fmt.Errorf("%w: has child teams: %v", consts.ErrWrongArg, children)
	}
	// TODO:
	// Check no teammates - checked by memdb engine
	// Check not default for any feature_flag
	// Delete all child IAM.group & IAM.rolebinding

	archivingTimestamp, archivingHash := ArchivingLabel()
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
	return s.repo.Restore(id)
}
