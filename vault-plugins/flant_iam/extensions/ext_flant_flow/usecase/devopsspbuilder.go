package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type devopsServicePackBuilder struct {
	identitySharingRepo   *iam_repo.IdentitySharingRepository
	roleBindingRepository *iam_repo.RoleBindingRepository
	teamRepo              *repo.TeamRepository
	servicePackRepo       *repo.ServicePackRepository
	liveConfig            *config.FlantFlowConfig
}

func (d devopsServicePackBuilder) OnCreateProject(project model.Project) error {
	if devopsCFG, err, ok := model.TryGetDevopsCFG(project.ServicePacks); ok {
		if err != nil {
			return err
		}
		// just create sharing if needs
		groups, is, err := d.createIdentitySharing(project.TenantUUID, d.liveConfig.FlantTenantUUID, devopsCFG.Team)
		if err != nil {
			return err
		}
		// create ssh rolebinding
		rbs, err := d.createRoleBindings(project.TenantUUID, project.UUID, groups, d.liveConfig.ServicePacksRolesSpecification[model.DevOps])
		if err != nil {
			return err
		}
		sp := model.ServicePack{
			ProjectUUID:      project.UUID,
			Name:             model.DevOps,
			Rolebindings:     rbs,
			IdentitySharings: []iam_model.IdentitySharingUUID{is.UUID},
		}
		return d.servicePackRepo.Create(&sp)
	}
	return nil
}

func (d devopsServicePackBuilder) OnUpdateProject(oldProject model.Project, updatedProject model.Project) error {
	oldDevopsCFG, err, oldCfgExists := model.TryGetDevopsCFG(oldProject.ServicePacks)
	if err != nil {
		return err
	}
	newDevopsCFG, err, newCfgExists := model.TryGetDevopsCFG(updatedProject.ServicePacks)
	if err != nil {
		return err
	}
	switch {
	case !newCfgExists && oldCfgExists:
		return d.OnDeleteProject(oldProject)
	case newCfgExists && !oldCfgExists:
		return d.OnCreateProject(updatedProject)
	case newCfgExists && oldCfgExists:
		{
			if *oldDevopsCFG == *newDevopsCFG {
				return nil
			}
			if err = d.OnDeleteProject(oldProject); err != nil {
				return err
			}
			return d.OnCreateProject(updatedProject)
		}
	}
	return nil
}

func (d devopsServicePackBuilder) OnDeleteProject(oldProject model.Project) error {
	archiveMark := memdb.NewArchiveMark()
	if _, err, ok := model.TryGetDevopsCFG(oldProject.ServicePacks); ok {
		if err != nil {
			return err
		}
		sp, err := d.servicePackRepo.GetByID(oldProject.UUID, model.DevOps)
		if err != nil {
			return err
		}
		// delete servicepack
		err = d.servicePackRepo.Delete(oldProject.UUID, model.DevOps, archiveMark)
		if err != nil {
			return err
		}
		// try delete rolbindings
		for _, rbUUID := range sp.Rolebindings {
			err := d.roleBindingRepository.CascadeDelete(rbUUID, archiveMark)
			if err != nil && !errors.Is(err, memdb.ErrNotEmptyRelation) {
				return err
			}
		}

		// try delete IdentitySharing
		for _, isUUID := range sp.IdentitySharings {
			if err = d.identitySharingRepo.Delete(isUUID, archiveMark); err != nil &&
				!errors.Is(err, memdb.ErrNotEmptyRelation) {
				sp1, _ := d.servicePackRepo.GetByID(sp.ProjectUUID, model.DevOps)
				println(err.Error())
				fmt.Printf("%#v\n", sp1)
				return err
			}
		}
	}
	return nil
}

func (d devopsServicePackBuilder) createIdentitySharing(clientTenantUUID iam_model.TenantUUID, flantTenantUUID iam_model.TenantUUID,
	teamUUID model.TeamUUID) ([]model.LinkedGroup, *iam_model.IdentitySharing, error) {
	team, err := d.teamRepo.GetByID(teamUUID)
	if err != nil {
		return nil, nil, err
	}

	identitySharings, err := d.identitySharingRepo.ListForDestinationTenant(clientTenantUUID)
	if err != nil {
		return nil, nil, err
	}
	groupsUUIDs := buildGroupUUIDs(team.Groups)
	sh := findEqualIdentitySharing(identitySharings, flantTenantUUID, groupsUUIDs)
	if sh != nil {
		return team.Groups, sh, nil
	}
	sh = &iam_model.IdentitySharing{
		UUID:                  uuid.New(),
		SourceTenantUUID:      flantTenantUUID,
		DestinationTenantUUID: clientTenantUUID,
		Version:               uuid.New(),
		Groups:                groupsUUIDs,
		Origin:                consts.OriginFlantFlow,
	}
	if err = d.identitySharingRepo.Create(sh); err != nil {
		return nil, nil, err
	}

	return team.Groups, sh, nil
}

func (d devopsServicePackBuilder) createRoleBindings(clientTenantUUID iam_model.TenantUUID, projectUUID iam_model.ProjectUUID,
	linkedGroups []model.LinkedGroup, rules map[model.LinkedGroupType][]iam_model.BoundRole) ([]iam_model.RoleBindingUUID, error) {
	var roleBindings []iam_model.RoleBindingUUID
	for ruleGroupType, boundRoles := range rules {
		var groupUUID iam_model.GroupUUID
		for _, linkedGroup := range linkedGroups {
			if linkedGroup.Type == ruleGroupType {
				groupUUID = linkedGroup.GroupUUID
			}
		}
		rb := &iam_model.RoleBinding{
			UUID:        uuid.New(),
			TenantUUID:  clientTenantUUID,
			Version:     uuid.New(),
			Description: model.DevOps,
			Groups:      []iam_model.GroupUUID{groupUUID},
			Members:     buildMembers(iam_model.GroupType, []iam_model.GroupUUID{groupUUID}),
			Projects:    []iam_model.ProjectUUID{projectUUID},
			Roles:       boundRoles,
			Origin:      consts.OriginFlantFlow,
			ValidTill:   0, // valid forever
		}
		err := d.roleBindingRepository.Create(rb)
		if err != nil {
			return nil, err
		}
		roleBindings = append(roleBindings, rb.UUID)
	}
	return roleBindings, nil
}

func buildMembers(memberType string, uuids []iam_model.GroupUUID) []iam_model.MemberNotation {
	members := make([]iam_model.MemberNotation, 0, len(uuids))
	for _, meberUUID := range uuids {
		members = append(members, iam_model.MemberNotation{
			Type: memberType,
			UUID: meberUUID,
		})
	}
	return members
}

func buildGroupUUIDs(groups []model.LinkedGroup) []iam_model.GroupUUID {
	result := make([]iam_model.GroupUUID, 0, len(groups))
	for _, g := range groups {
		result = append(result, g.GroupUUID)
	}
	return result
}

func findEqualIdentitySharing(identitySharings []*iam_model.IdentitySharing, sourceTenantUUID iam_model.TenantUUID,
	groups []iam_model.GroupUUID) *iam_model.IdentitySharing {
	groupUUIDs := map[iam_model.GroupUUID]struct{}{}
	for _, g := range groups {
		groupUUIDs[g] = struct{}{}
	}
	for _, is := range identitySharings {
		if is.Archived() ||
			is.SourceTenantUUID != sourceTenantUUID ||
			is.Origin != consts.OriginFlantFlow ||
			len(is.Groups) != len(groups) {
			continue
		}
		equal := true
		for _, g := range is.Groups {
			if _, ok := groupUUIDs[g]; !ok {
				equal = false
			}
		}
		if equal {
			return is
		}
	}
	return nil
}

func newDevopsServicePackBuilder(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) ServicePackController {
	return &devopsServicePackBuilder{
		identitySharingRepo:   iam_repo.NewIdentitySharingRepository(db),
		roleBindingRepository: iam_repo.NewRoleBindingRepository(db),
		teamRepo:              repo.NewTeamRepository(db),
		servicePackRepo:       repo.NewServicePackRepository(db),
		liveConfig:            liveConfig,
	}
}
