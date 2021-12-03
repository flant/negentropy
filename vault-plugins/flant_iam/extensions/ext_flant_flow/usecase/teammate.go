package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type TeammateService struct {
	flantTenantUUID iam_model.TenantUUID
	repo            *repo.TeammateRepository
	teamRepo        *repo.TeamRepository
	userService     *iam_usecase.UserService
}

func Teammates(db *io.MemoryStoreTxn, flantTenantUUID iam_model.TenantUUID) *TeammateService {
	return &TeammateService{
		flantTenantUUID: flantTenantUUID,
		repo:            repo.NewTeammateRepository(db),
		teamRepo:        repo.NewTeamRepository(db),
		userService:     iam_usecase.Users(db, flantTenantUUID, consts.OriginFlantFlow),
	}
}

func (s *TeammateService) Create(t *model.FullTeammate) error {
	teammate := t.GetTeammate()
	err := s.validateRole(teammate)
	if err != nil {
		return err
	}
	t.Version = repo.NewResourceVersion()
	err = s.userService.Create(&t.User)
	if err != nil {
		return err
	}
	teammate.Version = t.Version
	return s.repo.Create(teammate)
}

func (s *TeammateService) Update(updated *model.FullTeammate) error {
	teammate := updated.GetTeammate()
	if err := s.validateRole(teammate); err != nil {
		return err
	}
	stored, err := s.repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}
	if stored.RoleAtTeam != updated.RoleAtTeam {
		return fmt.Errorf("%w: role_at_team ", consts.ErrInvalidArg)
	}
	if stored.Version != updated.Version {
		return consts.ErrBadVersion
	}

	updated.Version = repo.NewResourceVersion()
	err = s.userService.Update(&updated.User)
	if err != nil {
		return err
	}
	// Update
	teammate.Version = updated.Version
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
	return s.repo.Delete(id, archiveMark)
}

func (s *TeammateService) GetByID(id iam_model.UserUUID) (*model.FullTeammate, error) {
	user, err := s.userService.GetByID(id)
	if err != nil {
		return nil, err
	}
	tm, err := s.repo.GetByID(id)
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
	return makeFullTeammate(user, tm)
}

func (s *TeammateService) validateRole(t *model.Teammate) error {
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
