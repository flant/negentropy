package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type GroupsController interface {
	// OnCreateTeammate : User should appear at all suitable teams in groups "SOMETYPE" type
	OnCreateTeammate(teammate model.Teammate) error

	// OnUpdateTeammate : if new TeamUUID differ from old one,
	// User should disappear at all suitable teams in groups "SOMETYPE" type, for old teamUUID and
	// User should appear at all suitable teams in groups "SOMETYPE" type, for new teamUUID
	// it is a combination of CreateTeammate & DeleteTeammate
	OnUpdateTeammate(oldTeammate model.Teammate, newTeammate model.Teammate) error

	// OnDeleteTeammate : // User should disappear at all suitable teams in groups "SOMETYPE" type
	OnDeleteTeammate(teammate model.Teammate) error

	// OnCreateTeam : Just create specific empty group
	OnCreateTeam(team model.Team) (model.Team, error)

	// OnUpdateTeam : If  new ParentTeamUUID differ from old one,
	// Specific users should disapper from suitable teams in groups "SOMETYPE" type
	// Specific users should apper in suitable teams in groups "SOMETYPE" type
	OnUpdateTeam(oldTeam model.Team, newTeam model.Team) (model.Team, error)

	// OnDeleteTeam : Just delete specific empty group (we can't delete not empty group)
	OnDeleteTeam(team model.Team) (model.Team, error)
}

type GroupsBuilder interface {
	// GroupType return specific SOMETYPE of group, which is controlled by concrete builder
	GroupType() string

	GroupsController
}

func GroupBuilders(db *io.MemoryStoreTxn, flantTenantUUID iam_model.TenantUUID) []GroupsBuilder {
	return []GroupsBuilder{newDirectBuilder(db, flantTenantUUID)}
}

const DirectMembersGroupType = "DIRECT"

type directBuilder struct {
	flantTenantUUID iam_model.TenantUUID
	groupService    *usecase.GroupService
	teamsRepo       *repo.TeamRepository
}

func newDirectBuilder(db *io.MemoryStoreTxn, flantTenantUUID iam_model.TenantUUID) *directBuilder {
	return &directBuilder{
		flantTenantUUID: flantTenantUUID,
		groupService:    usecase.Groups(db, flantTenantUUID, consts.OriginFlantFlow),
		teamsRepo:       repo.NewTeamRepository(db),
	}
}

func (d directBuilder) GroupType() string {
	return DirectMembersGroupType
}

func (d directBuilder) OnCreateTeammate(teammate model.Teammate) error {
	// User should appear at all suitable teams in groups "DIRECT" type
	team, err := d.teamsRepo.GetByID(teammate.TeamUUID)
	if err != nil {
		return err
	}
	suitableTeams := []model.Team{*team}
	return executeForEachTeamUnderSuitableGroup(suitableTeams,
		func(candidateGroup model.LinkedGroup) bool {
			return candidateGroup.Type == d.GroupType()
		},
		func(targetGroupUUID iam_model.GroupUUID) error {
			return d.groupService.AddUsersToGroup(targetGroupUUID, teammate.UserUUID)
		})
}

func executeForEachTeamUnderSuitableGroup(teams []model.Team, condition func(model.LinkedGroup) bool,
	handler func(iam_model.GroupUUID) error) error {
	for _, team := range teams {
		for _, g := range team.Groups {
			if condition(g) {
				err := handler(g.GroupUUID)
				if err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (d directBuilder) OnUpdateTeammate(oldTeammate model.Teammate, newTeammate model.Teammate) error {
	// OnUpdateTeammate : if new TeamUUID differ from old one,
	// User should disappear at all suitable teams in groups "SOMETYPE" type, for old teamUUID and
	// User should appear at all suitable teams in groups "SOMETYPE" type, for new teamUUID
	if oldTeammate.TeamUUID == newTeammate.TeamUUID {
		return nil
	}
	if err := d.OnCreateTeammate(newTeammate); err != nil {
		return err
	}
	if err := d.OnDeleteTeammate(oldTeammate); err != nil {
		return err
	}
	return nil
}

func (d directBuilder) OnDeleteTeammate(teammate model.Teammate) error {
	// User should disappear at all suitable teams in groups "DIRECT" type
	team, err := d.teamsRepo.GetByID(teammate.TeamUUID)
	if err != nil {
		return err
	}
	suitableTeams := []model.Team{*team}
	return executeForEachTeamUnderSuitableGroup(suitableTeams,
		func(candidateGroup model.LinkedGroup) bool {
			return candidateGroup.Type == d.GroupType()
		},
		func(targetGroupUUID iam_model.GroupUUID) error {
			return d.groupService.RemoveUsersFromGroup(targetGroupUUID, teammate.UserUUID)
		})
}

func (d directBuilder) OnCreateTeam(team model.Team) (model.Team, error) {
	g := &iam_model.Group{
		UUID:       uuid.New(),
		TenantUUID: d.flantTenantUUID,
		Identifier: d.GroupType() + "_" + team.Identifier,
	}
	err := d.groupService.Create(g)
	if err != nil {
		return team, err
	}
	team.Groups = append(team.Groups, model.LinkedGroup{
		GroupUUID: g.UUID,
		Type:      d.GroupType(),
	})
	return team, nil
}

func (d directBuilder) OnUpdateTeam(oldTeam model.Team, newTeam model.Team) (model.Team, error) {
	// OnUpdateTeam : If  new ParentTeamUUID differ from old one,
	// Specific users should disapper from suitable teams in groups "SOMETYPE" type
	// Specific users should apper in suitable teams in groups "SOMETYPE" type
	// For DIRECT type = do nothing
	return newTeam, nil
}

func (d directBuilder) OnDeleteTeam(team model.Team) (model.Team, error) {
	targetIdx := -1
	for i := range team.Groups {
		if team.Groups[i].Type == d.GroupType() {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return team, nil
	}
	groupUUID := team.Groups[targetIdx].GroupUUID
	err := d.groupService.Delete(groupUUID)
	if err != nil {
		return team, err
	}
	groups := team.Groups
	team.Groups = append(groups[:targetIdx], groups[targetIdx+1:]...) // nolint:gocritic
	return team, nil
}

type groupsController struct {
	groupBuilders []GroupsBuilder
}

func (g groupsController) OnCreateTeammate(teammate model.Teammate) error {
	for _, c := range g.groupBuilders {
		err := c.OnCreateTeammate(teammate)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g groupsController) OnUpdateTeammate(oldTeammate model.Teammate, newTeammate model.Teammate) error {
	for _, c := range g.groupBuilders {
		err := c.OnUpdateTeammate(oldTeammate, newTeammate)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g groupsController) OnDeleteTeammate(teammate model.Teammate) error {
	for _, c := range g.groupBuilders {
		err := c.OnDeleteTeammate(teammate)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g groupsController) OnCreateTeam(team model.Team) (model.Team, error) {
	var err error
	for _, c := range g.groupBuilders {
		team, err = c.OnCreateTeam(team)
		if err != nil {
			return team, err
		}
	}
	return team, nil
}

func (g groupsController) OnUpdateTeam(oldTeam model.Team, newTeam model.Team) (model.Team, error) {
	var err error
	for _, c := range g.groupBuilders {
		newTeam, err = c.OnUpdateTeam(oldTeam, newTeam)
		if err != nil {
			return newTeam, err
		}
	}
	return newTeam, err
}

func (g groupsController) OnDeleteTeam(team model.Team) (model.Team, error) {
	var err error
	for _, c := range g.groupBuilders {
		team, err = c.OnDeleteTeam(team)
		if err != nil {
			return team, err
		}
	}
	return team, err
}

func NewGroupsController(db *io.MemoryStoreTxn, flantTenantUUID iam_model.TenantUUID) GroupsController {
	return groupsController{
		groupBuilders: []GroupsBuilder{
			newDirectBuilder(db, flantTenantUUID),
		},
	}
}
