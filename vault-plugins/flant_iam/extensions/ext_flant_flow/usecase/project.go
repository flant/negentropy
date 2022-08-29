package usecase

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ProjectService struct {
	*iam_usecase.ProjectService
	teamRepo               *repo.TeamRepository
	servicePacksController ServicePackController
	liveConfig             *config.FlantFlowConfig
}

func Projects(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) *ProjectService {
	return &ProjectService{
		ProjectService:         iam_usecase.Projects(db, consts.OriginFlantFlow),
		teamRepo:               repo.NewTeamRepository(db),
		servicePacksController: NewServicePackController(db, liveConfig),
		liveConfig:             liveConfig,
	}
}

type ProjectParams struct {
	IamProject              *iam.Project
	ServicePackNames        map[model.ServicePackName]struct{}
	DevopsTeamUUID          model.TeamUUID
	InternalProjectTeamUUID model.TeamUUID
	ConsultingTeamUUID      model.TeamUUID
}

// build servicePacks with CFGs
func (s *ProjectService) buildServicePacks(params ProjectParams) (map[model.ServicePackName]model.ServicePackCFG, error) {
	servicepacks := map[model.ServicePackName]model.ServicePackCFG{}
	for spn := range params.ServicePackNames {
		switch spn {
		case model.DevOps:
			if params.DevopsTeamUUID == "" {
				return nil, fmt.Errorf("%w: service_pack %q needs passed devops_team", consts.ErrInvalidArg, spn)
			}
			if team, err := s.teamRepo.GetByID(params.DevopsTeamUUID); err != nil {
				return nil, fmt.Errorf("service_pack %s: devops_team: %s:%w", spn, params.DevopsTeamUUID, err)
			} else if team.TeamType != model.DevopsTeam {
				return nil, fmt.Errorf("%w: service_pack %s: wrong passed team type: %s", consts.ErrInvalidArg, spn, team.TeamType)
			}
			servicepacks[spn] = model.DevopsServicePackCFG{
				Team: params.DevopsTeamUUID,
			}
		case model.InternalProject:
			if params.IamProject.TenantUUID != s.liveConfig.FlantTenantUUID {
				return nil, fmt.Errorf("%w: service_pack %s is allowed only for flant internal projects", consts.ErrInvalidArg, model.InternalProject)
			}
			if params.InternalProjectTeamUUID == "" {
				return nil, fmt.Errorf("%w: service_pack %q needs passed team", consts.ErrInvalidArg, spn)
			}
			if _, err := s.teamRepo.GetByID(params.InternalProjectTeamUUID); err != nil {
				return nil, fmt.Errorf(" service_pack %s: team: %s:%w", spn, params.InternalProjectTeamUUID, err)
			}
			servicepacks[spn] = model.InternalProjectServicePackCFG{
				Team: params.InternalProjectTeamUUID,
			}
		case model.Consulting:
			if params.ConsultingTeamUUID == "" {
				return nil, fmt.Errorf("%w: service_pack %q needs passed consulting_team", consts.ErrInvalidArg, spn)
			}
			if _, err := s.teamRepo.GetByID(params.ConsultingTeamUUID); err != nil {
				return nil, fmt.Errorf("service_pack %s: consulting_team: %s:%w", spn, params.ConsultingTeamUUID, err)
			}
			servicepacks[spn] = model.ConsultingServicePackCFG{
				Team: params.ConsultingTeamUUID,
			}

		default:
			servicepacks[spn] = s.buildServicePackCfgByName(spn)
		}
	}
	if len(servicepacks) == 0 {
		return nil, fmt.Errorf("%w:empty service_packs", consts.ErrInvalidArg)
	}
	return servicepacks, nil
}

func (s *ProjectService) Create(projectParams ProjectParams) (*model.Project, error) {
	servicePacks, err := s.buildServicePacks(projectParams)
	if err != nil {
		return nil, err
	}
	iamProject := updateExtensions(*projectParams.IamProject, servicePacks)
	iamProject.Origin = consts.OriginFlantFlow

	if err := s.ProjectService.Create(&iamProject); err != nil {
		return nil, err
	}
	project, err := makeProject(&iamProject)
	if err != nil {
		return nil, err
	}
	if err = s.servicePacksController.OnCreateProject(*project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) Update(projectParams ProjectParams) (*model.Project, error) {
	stored, err := s.ProjectService.GetByID(projectParams.IamProject.UUID)
	if stored.TenantUUID != projectParams.IamProject.TenantUUID {
		return nil, consts.ErrNotFound
	}
	if stored.Origin != consts.OriginFlantFlow {
		return nil, consts.ErrBadOrigin
	}
	if stored.Version != projectParams.IamProject.Version {
		return nil, consts.ErrBadVersion
	}
	if stored.Archived() {
		return nil, consts.ErrIsArchived
	}
	servicePacks, err := s.buildServicePacks(projectParams)
	if err != nil {
		return nil, err
	}
	iamProject := updateExtensions(*projectParams.IamProject, servicePacks)
	iamProject.Origin = consts.OriginFlantFlow
	project, err := makeProject(&iamProject)
	if err != nil {
		return nil, err
	}
	oldProject, err := makeProject(stored)
	if err != nil {
		return nil, err
	}
	err = s.servicePacksController.OnUpdateProject(*oldProject, *project)
	if err != nil {
		return nil, err
	}
	err = s.ProjectService.Update(&iamProject)
	if err != nil {
		return nil, err
	}
	project.Version = iamProject.Version
	return project, nil
}

func (s *ProjectService) Delete(id model.ProjectUUID) error {
	iamProject, err := s.ProjectService.GetByID(id)
	if err != nil {
		return err
	}
	project, err := makeProject(iamProject)
	if err != nil {
		return err
	}
	if err := s.servicePacksController.OnDeleteProject(*project); err != nil {
		return err
	}
	return s.ProjectService.Delete(id)
}

func (s *ProjectService) List(cid model.ClientUUID, showArchived bool) ([]*model.Project, error) {
	iamProjects, err := s.ProjectService.List(cid, showArchived)
	if err != nil {
		return nil, err
	}
	result := make([]*model.Project, 0, len(iamProjects))
	for i := range iamProjects {
		if iamProjects[i].Origin == consts.OriginFlantFlow {
			p, err := makeProject(iamProjects[i])
			if err != nil {
				return nil, err
			}
			result = append(result, p)
		}
	}
	return result, nil
}

func makeProject(project *iam.Project) (*model.Project, error) {
	if project == nil {
		return nil, consts.ErrNilPointer
	}
	var servicePacks map[model.ServicePackName]model.ServicePackCFG
	if exts := project.Extensions; exts != nil {
		if ext := exts[consts.OriginFlantFlow]; ext != nil {
			if ext.OwnerType == iam.ProjectType && ext.OwnerUUID == project.UUID {
				var err error
				servicePacks, err = unmarshallServicePackCandidate(ext.Attributes["service_packs"])
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return &model.Project{
		ArchiveMark:  project.ArchiveMark,
		UUID:         project.UUID,
		TenantUUID:   project.TenantUUID,
		Version:      project.Version,
		Identifier:   project.Identifier,
		FeatureFlags: project.FeatureFlags,
		Origin:       "",
		ServicePacks: servicePacks,
	}, nil
}

func unmarshallServicePackCandidate(servicePacksRaw interface{}) (map[model.ServicePackName]model.ServicePackCFG, error) {
	var servicePacks map[model.ServicePackName]model.ServicePackCFG
	var err error
	switch spMap := servicePacksRaw.(type) {
	// after kafka restoration
	case map[model.ServicePackName]interface{}:
		servicePacks, err = model.ParseServicePacks(spMap)
		if err != nil {
			return nil, err
		}
	case map[model.ServicePackName]model.ServicePackCFG:
		servicePacks = spMap
	default:
		return nil, fmt.Errorf("%w:need map[string]interface{} or map[model.ServicePackName]model.ServicePackCFG, passed:%T",
			consts.ErrWrongType, servicePacksRaw)
	}
	return servicePacks, nil
}

func makeIamProject(project *model.Project) (*iam.Project, error) {
	if project == nil {
		return nil, consts.ErrNilPointer
	}
	iamProject := iam.Project{
		ArchiveMark:  project.ArchiveMark,
		UUID:         project.UUID,
		TenantUUID:   project.TenantUUID,
		Version:      project.Version,
		Identifier:   project.Identifier,
		FeatureFlags: project.FeatureFlags,
		Origin:       project.Origin,
	}
	iamProject = updateExtensions(iamProject, project.ServicePacks)
	return &iamProject, nil
}

func updateExtensions(iamProject iam.Project, servicePacks map[model.ServicePackName]model.ServicePackCFG) iam.Project {
	if iamProject.Extensions == nil {
		iamProject.Extensions = map[consts.ObjectOrigin]*iam.Extension{}
	}
	iamProject.Extensions[consts.OriginFlantFlow] = &iam.Extension{
		Origin:    consts.OriginFlantFlow,
		OwnerType: iam.ProjectType,
		OwnerUUID: iamProject.UUID,
		Attributes: map[string]interface{}{
			"service_packs": servicePacks,
		},
	}
	return iamProject
}

func (s *ProjectService) GetByID(pid model.ProjectUUID) (*model.Project, error) {
	iamProject, err := s.ProjectService.GetByID(pid)
	if err != nil {
		return nil, err
	}
	if iamProject.Origin != consts.OriginFlantFlow {
		return nil, consts.ErrBadOrigin
	}
	return makeProject(iamProject)
}

func (s *ProjectService) Restore(pid model.ProjectUUID) (*model.Project, error) {
	iamProject, err := s.ProjectService.Restore(pid)
	if err != nil {
		return nil, err
	}
	return makeProject(iamProject)
}

func (s *ProjectService) buildServicePackCfgByName(spn model.ServicePackName) model.ServicePackCFG {
	switch spn {
	case model.L1:
		return model.L1ServicePackCFG{Team: s.liveConfig.SpecificTeams[config.L1]}
	case model.Mk8s:
		return model.Mk8sServicePackCFG{Team: s.liveConfig.SpecificTeams[config.Mk8s]}
	case model.Okmeter:
		return model.OkmeterServicePackCFG{Team: s.liveConfig.SpecificTeams[config.Okmeter]}
	case model.Deckhouse:
		return model.DeckhouseServicePackCFG{Team: s.liveConfig.SpecificTeams[config.Mk8s]}
	}
	return nil
}
