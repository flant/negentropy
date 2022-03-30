package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type TeammateService struct {
	liveConfig       *config.FlantFlowConfig
	repo             *repo.TeammateRepository
	teamRepo         *repo.TeamRepository
	groupRepo        *iam_repo.GroupRepository
	userService      *iam_usecase.UserService
	groupsController GroupsController
}

func Teammates(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) *TeammateService {
	return &TeammateService{
		liveConfig:       liveConfig,
		repo:             repo.NewTeammateRepository(db),
		teamRepo:         repo.NewTeamRepository(db),
		groupRepo:        iam_repo.NewGroupRepository(db),
		userService:      iam_usecase.Users(db, liveConfig.FlantTenantUUID, consts.OriginFlantFlow),
		groupsController: NewGroupsController(db, liveConfig.FlantTenantUUID),
	}
}

func (s *TeammateService) Create(t *model.FullTeammate) error {
	teammate := t.ExtractTeammate()
	err := s.validateTeamRole(teammate)
	if err != nil {
		return err
	}
	t.Version = repo.NewResourceVersion()
	err = s.userService.Create(&t.User)
	if err != nil {
		return err
	}
	teammate.Version = t.Version
	err = s.groupsController.OnCreateTeammate(*teammate)
	if err != nil {
		return err
	}
	err = s.repo.Create(teammate)
	if err != nil {
		return err
	}

	return s.addTeammateToFlantAllGroup(teammate)
}

func (s *TeammateService) addTeammateToFlantAllGroup(teammate *model.Teammate) error {
	flantAllGroup, err := s.groupRepo.GetByID(s.liveConfig.AllFlantGroup)
	if err != nil {
		return err
	}
	flantAllGroup.Users = append(flantAllGroup.Users, teammate.UserUUID)
	flantAllGroup.Members = append(flantAllGroup.Members, iam_model.MemberNotation{
		Type: "user",
		UUID: teammate.UserUUID,
	})
	return s.groupRepo.Update(flantAllGroup)
}

func (s *TeammateService) Update(updated *model.FullTeammate) error {
	teammate := updated.ExtractTeammate()
	if err := s.validateTeamRole(teammate); err != nil {
		return err
	}
	stored, err := s.repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}
	if stored.Version != updated.Version {
		return consts.ErrBadVersion
	}

	err = s.userService.Update(&updated.User)
	if err != nil {
		return err
	}
	// Update
	teammate.Version = updated.Version
	err = s.groupsController.OnUpdateTeammate(*stored, *teammate)
	if err != nil {
		return err
	}
	return s.repo.Update(teammate)
}

func (s *TeammateService) Delete(id iam_model.UserUUID) error {
	err := s.userService.Delete(id)
	if err != nil {
		return err
	}
	user, err := s.userService.GetByID(id)
	if err != nil {
		return err
	}
	archiveMark := user.ArchiveMark
	stored, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	err = s.groupsController.OnDeleteTeammate(*stored)
	if err != nil {
		return err
	}
	err = s.repo.Delete(id, archiveMark)
	if err != nil {
		return err
	}
	// err = s.removeTeammateFromFlantAllGroup(stored) // should deleted by shared.memdb mechanics
	return nil
}

func (s *TeammateService) GetByID(id iam_model.UserUUID, teamUUID model.TeamUUID) (*model.FullTeammate, error) {
	tm, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if tm.TeamUUID != teamUUID {
		return nil, consts.ErrNotFound
	}
	user, err := s.userService.GetByID(id)
	if err != nil {
		return nil, err
	}

	return makeFullTeammate(user, tm)
}

func (s *TeammateService) List(teamID model.TeamUUID, showArchived bool) ([]*model.FullTeammate, error) {
	tms, err := s.repo.List(teamID, showArchived)
	if err != nil {
		return nil, err
	}
	result := make([]*model.FullTeammate, len(tms))
	for i := range tms {
		user, err := s.userService.GetByID(tms[i].UserUUID)
		if err != nil {
			return nil, err
		}
		result[i], err = makeFullTeammate(user, tms[i])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *TeammateService) Restore(id iam_model.UserUUID) (*model.FullTeammate, error) {
	user, err := s.userService.Restore(id)
	if err != nil {
		return nil, err
	}
	tm, err := s.repo.Restore(id)
	if err != nil {
		return nil, err
	}
	err = s.groupsController.OnCreateTeammate(*tm)
	if err != nil {
		return nil, err
	}
	fullTeammate, err := makeFullTeammate(user, tm)
	if err != nil {
		return nil, err
	}
	err = s.addTeammateToFlantAllGroup(fullTeammate.ExtractTeammate())
	if err != nil {
		return nil, err
	}
	return fullTeammate, nil
}

func (s *TeammateService) validateTeamRole(t *model.Teammate) error {
	team, err := s.teamRepo.GetByID(t.TeamUUID)
	if errors.Is(err, consts.ErrNotFound) {
		return fmt.Errorf("%w: team with uuid:%s not found", consts.ErrInvalidArg, t.TeamUUID)
	}
	if err != nil {
		return err
	}
	if _, ok := model.TeamRoles[team.TeamType][t.RoleAtTeam]; !ok {
		return fmt.Errorf("%w: role %s is not allowed for %s team", consts.ErrInvalidArg, t.RoleAtTeam, team.TeamType)
	}
	return nil
}

func makeFullTeammate(user *iam_model.User, tm *model.Teammate) (*model.FullTeammate, error) {
	if user == nil || tm == nil {
		return nil, consts.ErrNilPointer
	}
	return &model.FullTeammate{
		User:       *user,
		TeamUUID:   tm.TeamUUID,
		RoleAtTeam: tm.RoleAtTeam,
	}, nil
}
