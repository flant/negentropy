package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type TeamService struct {
	flantTenantUUID  iam_model.TenantUUID
	repo             *repo.TeamRepository
	teammateRepo     *repo.TeammateRepository
	groupsController GroupsController
	liveConfig       *config.FlantFlowConfig
}

func Teams(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) *TeamService {
	return &TeamService{
		flantTenantUUID:  liveConfig.FlantTenantUUID,
		repo:             repo.NewTeamRepository(db),
		teammateRepo:     repo.NewTeammateRepository(db),
		groupsController: NewGroupsController(db, liveConfig.FlantTenantUUID),
		liveConfig:       liveConfig,
	}
}

func (s *TeamService) Create(t *model.Team) error {
	_, err := s.repo.GetByIdentifier(t.Identifier)
	if !errors.Is(err, consts.ErrNotFound) {
		return fmt.Errorf("%w: %s", consts.ErrAlreadyExists, t.Identifier)
	}
	err = s.validateParentTeamUUID(t)
	if err != nil {
		return err
	}
	if _, allowed := model.TeamTypes[t.TeamType]; !allowed {
		return fmt.Errorf("%w: team_type: '%s' is not allowed", consts.ErrInvalidArg, t.TeamType)
	}
	t.Version = repo.NewResourceVersion()
	*t, err = s.groupsController.OnCreateTeam(*t)
	if err != nil {
		return err
	}
	return s.repo.Create(t)
}

func (s *TeamService) validateParentTeamUUID(t *model.Team) error {
	if t.ParentTeamUUID != "" {
		_, err := s.repo.GetByID(t.ParentTeamUUID)
		if errors.Is(err, consts.ErrNotFound) {
			return fmt.Errorf("%w: parent_team_uuid must be valid team uuid or empty", consts.ErrInvalidArg)
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
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	updated.TeamType = stored.TeamType // team_type cant't be changed
	if stored.Version != updated.Version {
		return consts.ErrBadVersion
	}
	updated.Version = repo.NewResourceVersion()
	*updated, err = s.groupsController.OnUpdateTeam(*stored, *updated)
	if err != nil {
		return err
	}
	return s.repo.Create(updated)
}

func (s *TeamService) Delete(id model.TeamUUID) error {
	err := checkTeamNotInConfig(id, s.liveConfig.SpecificTeams)
	if err != nil {
		return fmt.Errorf("%w:%s", consts.ErrInvalidArg, err.Error())
	}
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
		return fmt.Errorf("%w: has child teams: %v", consts.ErrInvalidArg, children)
	}
	// Check no teammates
	teammates, err := s.teammateRepo.List(id, false)
	if err != nil {
		return err
	}
	if len(teammates) > 0 {
		return fmt.Errorf("%w: team has %d teammates on board", consts.ErrInvalidArg, len(teammates))
	}
	// TODO:
	// Check not default for any feature_flag
	// Delete all child IAM.group & - deleted by GroupBuilder
	// Delete IAM.rolebinding
	archiveMark := memdb.NewArchiveMark()
	*team, err = s.groupsController.OnDeleteTeam(*team)
	if err != nil {
		return err
	}
	return s.repo.Delete(id, archiveMark)
}

func checkTeamNotInConfig(teamUUID string, teams map[config.SpecializedTeam]model.TeamUUID) error {
	for specification, specificTeamUUID := range teams {
		if specificTeamUUID == teamUUID {
			return fmt.Errorf("team %s is %s team", teamUUID, specification)
		}
	}
	return nil
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
