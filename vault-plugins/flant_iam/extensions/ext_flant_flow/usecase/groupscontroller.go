package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
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

type GroupsControllerProvider func(db *io.MemoryStoreTxn, flantTenantUUID iam_model.TenantUUID) GroupsBuilder

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

func NewGroupsController(db *io.MemoryStoreTxn, flantTenantUUID iam_model.TenantUUID) GroupsController {
	return groupsController{
		groupBuilders: []GroupsBuilder{
			newDirectBuilder(db, flantTenantUUID),
			newDirectManagersBuilder(db, flantTenantUUID),
			newManagersBuilder(db, flantTenantUUID),
		},
	}
}
