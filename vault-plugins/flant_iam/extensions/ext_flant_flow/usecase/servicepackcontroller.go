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

type ServicePackController interface {
	// OnCreateProject : analyze ServicePackCFG, create rolebindings and identitySharings, store servicePacks
	OnCreateProject(model.Project) error
	// OnUpdateProject : analyze ServicePackCFG changes, update and store serrvicePacks (rolebindings and identity sharings)
	OnUpdateProject(oldProject model.Project, updatedProject model.Project) error
	// OnDeleteProject : do nothing, as now Cascade deleting is used for project deletions
	OnDeleteProject(oldProject model.Project) error
}

type servicePackController struct {
	controllers []ServicePackController
}

func (s servicePackController) OnCreateProject(project model.Project) error {
	for _, c := range s.controllers {
		if err := c.OnCreateProject(project); err != nil {
			return err
		}
	}
	return nil
}

func (s servicePackController) OnUpdateProject(oldProject model.Project, updatedProject model.Project) error {
	for _, c := range s.controllers {
		if err := c.OnUpdateProject(oldProject, updatedProject); err != nil {
			return err
		}
	}
	return nil
}

func (s servicePackController) OnDeleteProject(oldProject model.Project) error {
	for _, c := range s.controllers {
		if err := c.OnCreateProject(oldProject); err != nil {
			return err
		}
	}
	return nil
}

func NewServicePackController(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) ServicePackController {
	return &servicePackController{
		[]ServicePackController{
			newDevopsServicePackBuilder(db, liveConfig),
		},
	}
}

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
		groupUUIDs, is, err := d.createIdentitySharing(project.TenantUUID, d.liveConfig.FlantTenantUUID, devopsCFG.DevopsTeam)
		if err != nil {
			return err
		}
		// create ssh rolebinding
		rb, err := d.createRoleBinding(project.TenantUUID, project.UUID, groupUUIDs, d.liveConfig.RolesForSpecificTeams[config.Devops])
		if err != nil {
			return err
		}
		sp := model.ServicePack{
			ProjectUUID:      project.UUID,
			Name:             model.DevOps,
			Version:          uuid.New(),
			Rolebindings:     []iam_model.RoleBindingUUID{rb.UUID},
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
	teamUUID model.TeamUUID) ([]iam_model.GroupUUID, *iam_model.IdentitySharing, error) {
	team, err := d.teamRepo.GetByID(teamUUID)
	if err != nil {
		return nil, nil, err
	}

	identitySharings, err := d.identitySharingRepo.ListForDestinationTenant(clientTenantUUID)
	if err != nil {
		return nil, nil, err
	}
	groups := buildGroupUUIDs(team.Groups)
	sh := findEqualIdentitySharing(identitySharings, flantTenantUUID, groups)
	if sh != nil {
		return groups, sh, nil
	}
	sh = &iam_model.IdentitySharing{
		UUID:                  uuid.New(),
		SourceTenantUUID:      flantTenantUUID,
		DestinationTenantUUID: clientTenantUUID,
		Version:               uuid.New(),
		Groups:                groups,
	}
	if err = d.identitySharingRepo.Create(sh); err != nil {
		return nil, nil, err
	}

	return groups, sh, nil
}

func (d devopsServicePackBuilder) createRoleBinding(clientTenantUUID iam_model.TenantUUID, projectUUID iam_model.ProjectUUID,
	groups []iam_model.GroupUUID, roles []iam_model.RoleName) (*iam_model.RoleBinding, error) {
	rbsOfProject, err := d.roleBindingRepository.FindDirectRoleBindingsForProject(projectUUID)
	if err != nil {
		return nil, err
	}
	rbsOfRole, err := d.roleBindingRepository.FindDirectRoleBindingsForRoles(roles...)
	filteredRoleBindings := map[iam_model.RoleBindingUUID]*iam_model.RoleBinding{}
	for uuid, rb := range rbsOfProject {
		if _, ok := rbsOfRole[uuid]; ok {
			filteredRoleBindings[uuid] = rb
		}
	}
	rb := findEqualRoleBinding(filteredRoleBindings, groups, roles)
	if rb != nil {
		return rb, nil
	}
	boundRoles := make([]iam_model.BoundRole, 0, len(roles))
	for _, role := range roles {
		boundRoles = append(boundRoles, iam_model.BoundRole{Name: role, Options: map[string]interface{}{}})
	}

	rb = &iam_model.RoleBinding{
		UUID:       uuid.New(),
		TenantUUID: clientTenantUUID,
		Version:    uuid.New(),
		Identifier: model.DevOps,
		Groups:     groups,
		Members:    buildMemebers(groups),
		Projects:   []iam_model.ProjectUUID{projectUUID},
		Roles:      boundRoles,
		Origin:     consts.OriginFlantFlow,
	}
	err = d.roleBindingRepository.Create(rb)
	if err != nil {
		return nil, err
	}
	return rb, nil
}

func buildMemebers(groups []iam_model.GroupUUID) []iam_model.MemberNotation {
	members := make([]iam_model.MemberNotation, 0, len(groups))
	for _, groupUUID := range groups {
		members = append(members, iam_model.MemberNotation{
			Type: iam_model.GroupType,
			UUID: groupUUID,
		})
	}
	return members
}

func findEqualRoleBinding(roleBindings map[iam_model.RoleBindingUUID]*iam_model.RoleBinding, groups []iam_model.GroupUUID,
	roles []iam_model.RoleName) *iam_model.RoleBinding {
	groupUUIDs := map[iam_model.GroupUUID]struct{}{}
	for _, g := range groups {
		groupUUIDs[g] = struct{}{}
	}
	roleNames := map[iam_model.RoleName]struct{}{}
	for _, r := range roles {
		roleNames[r] = struct{}{}
	}
	for _, rb := range roleBindings {
		if rb.Archived() ||
			len(rb.Groups) != len(groups) {
			continue
		}
		equal := true
		for _, g := range rb.Groups {
			if _, ok := groupUUIDs[g]; !ok {
				equal = false
				break
			}
			if equal {
				for _, r := range rb.Roles {
					if _, ok := roleNames[r.Name]; !ok {
						equal = false
						break
					}
				}
			}
			if equal {
				return rb
			}
		}
	}
	return nil
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
