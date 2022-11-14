package usecase

import (
	"errors"

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

type internalProjectServicePackBuilder struct {
	identitySharingRepo   *iam_repo.IdentitySharingRepository
	roleBindingRepository *iam_repo.RoleBindingRepository
	teamRepo              *repo.TeamRepository
	servicePackRepo       *repo.ServicePackRepository
	liveConfig            *config.FlantFlowConfig
}

func (d internalProjectServicePackBuilder) OnCreateProject(project model.Project) error {
	if cfg, err, ok := model.TryGetInternalProjectCFG(project.ServicePacks); ok {
		if err != nil {
			return err
		}
		team, err := d.teamRepo.GetByID(cfg.Team)
		if err != nil {
			return err
		}
		// create ssh rolebinding
		rbs, err := d.createRoleBindings(project.TenantUUID, project.UUID, team.Groups, d.liveConfig.ServicePacksRolesSpecification[model.InternalProject])
		if err != nil {
			return err
		}
		sp := model.ServicePack{
			ProjectUUID:      project.UUID,
			Name:             model.InternalProject,
			Rolebindings:     rbs,
			IdentitySharings: nil,
		}
		return d.servicePackRepo.Create(&sp)
	}
	return nil
}

func (d internalProjectServicePackBuilder) OnUpdateProject(oldProject model.Project, updatedProject model.Project) error {
	oldCfg, err, oldCfgExists := model.TryGetInternalProjectCFG(oldProject.ServicePacks)
	if err != nil {
		return err
	}
	newCfg, err, newCfgExists := model.TryGetInternalProjectCFG(updatedProject.ServicePacks)
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
			if *oldCfg == *newCfg {
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

func (d internalProjectServicePackBuilder) OnDeleteProject(oldProject model.Project) error {
	archiveMark := memdb.NewArchiveMark()
	if _, err, ok := model.TryGetInternalProjectCFG(oldProject.ServicePacks); ok {
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
	}
	return nil
}

func (d internalProjectServicePackBuilder) createRoleBindings(clientTenantUUID iam_model.TenantUUID, projectUUID iam_model.ProjectUUID,
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
			Description: model.InternalProject,
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

func newInternalProjectServicePackBuilder(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) ServicePackController {
	return &internalProjectServicePackBuilder{
		identitySharingRepo:   iam_repo.NewIdentitySharingRepository(db),
		roleBindingRepository: iam_repo.NewRoleBindingRepository(db),
		teamRepo:              repo.NewTeamRepository(db),
		servicePackRepo:       repo.NewServicePackRepository(db),
		liveConfig:            liveConfig,
	}
}
