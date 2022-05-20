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

// this group should just accumulate direct_managers_groups from this and all parents groups

const ManagersGroupType = "managers"

type managersBuilder struct {
	flantTenantUUID iam_model.TenantUUID
	groupService    *usecase.GroupService
	teamsRepo       *repo.TeamRepository
}

func newManagersBuilder(db *io.MemoryStoreTxn, flantTenantUUID iam_model.TenantUUID) GroupsBuilder {
	return &managersBuilder{
		flantTenantUUID: flantTenantUUID,
		groupService:    usecase.Groups(db, flantTenantUUID, consts.OriginFlantFlow),
		teamsRepo:       repo.NewTeamRepository(db),
	}
}

func (d managersBuilder) GroupType() string {
	return ManagersGroupType
}

func (d managersBuilder) OnCreateTeammate(teammate model.Teammate) error {
	// do nothing for managers type
	return nil
}

func (d managersBuilder) OnUpdateTeammate(oldTeammate model.Teammate, newTeammate model.Teammate) error {
	// do nothing for managers type
	return nil
}

func (d managersBuilder) OnDeleteTeammate(teammate model.Teammate) error {
	// do nothing for managers type
	return nil
}

func (d managersBuilder) OnCreateTeam(team model.Team) (model.Team, error) {
	groups, err := d.collectAllDirectManagersGroups(team)
	if err != nil {
		return team, err
	}
	g := &iam_model.Group{
		UUID:       uuid.New(),
		TenantUUID: d.flantTenantUUID,
		Identifier: team.Identifier + "_" + d.GroupType(),
		Groups:     groups,
		Members:    buildMembers(groups),
	}
	err = d.groupService.Create(g)
	if err != nil {
		return team, err
	}
	team.Groups = append(team.Groups, model.LinkedGroup{
		GroupUUID: g.UUID,
		Type:      d.GroupType(),
	})
	return team, nil
}

// collectAllDirectManagersGroups recursively collect directManagersGroups
func (d managersBuilder) collectAllDirectManagersGroups(team model.Team) ([]iam_model.GroupUUID, error) {
	var ownDirectManagersGroupUUID iam_model.GroupUUID
	for _, gr := range team.Groups {
		if gr.Type == DirectManagersGroupType {
			ownDirectManagersGroupUUID = gr.GroupUUID
			break
		}
	}
	var parentsDirectManagersGroupUUIDs []iam_model.GroupUUID
	if team.ParentTeamUUID != "" {
		parentTeam, err := d.teamsRepo.GetByID(team.ParentTeamUUID)
		if err != nil {
			return nil, err
		}
		parentsDirectManagersGroupUUIDs, err = d.collectAllDirectManagersGroups(*parentTeam)
	}
	result := append(parentsDirectManagersGroupUUIDs, ownDirectManagersGroupUUID)
	return result, nil
}

func (d managersBuilder) OnUpdateTeam(oldTeam model.Team, newTeam model.Team) (model.Team, error) {
	if newTeam.ParentTeamUUID != oldTeam.ParentTeamUUID {
		return d.OnCreateTeam(newTeam)
	} else {
		return newTeam, nil
	}
}

func (d managersBuilder) OnDeleteTeam(team model.Team) (model.Team, error) {
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
