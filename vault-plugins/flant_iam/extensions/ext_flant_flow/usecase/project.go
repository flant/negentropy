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
}

func Projects(db *io.MemoryStoreTxn, liveConfig *config.FlantFlowConfig) *ProjectService {
	return &ProjectService{
		ProjectService:         iam_usecase.Projects(db, consts.OriginFlantFlow),
		teamRepo:               repo.NewTeamRepository(db),
		servicePacksController: NewServicePackController(db, liveConfig),
	}
}

type ProjectParams struct {
	IamProject       *iam.Project
	ServicePackNames map[model.ServicePackName]struct{}
	DevopsTeamUUID   model.TeamUUID
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
				DevopsTeam: params.DevopsTeamUUID,
			}
		// TODO: others
		default:
			servicepacks[spn] = nil
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
	project := &model.Project{
		Project:      *projectParams.IamProject,
		ServicePacks: servicePacks,
	}
	iamProject, err := makeIamProject(project)
	if err != nil {
		return nil, err
	}
	if err := s.ProjectService.Create(iamProject); err != nil {
		return nil, err
	}
	iamProject, err = s.ProjectService.GetByID(project.UUID)
	if err != nil {
		return nil, err
	}
	project.Project = *iamProject
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
	project := &model.Project{
		Project:      *projectParams.IamProject,
		ServicePacks: servicePacks,
	}
	project.Extensions = stored.Extensions
	iamProject, err := makeIamProject(project)
	if err != nil {
		return nil, err
	}
	project.Project = *iamProject
	oldProject, err := makeProject(stored)
	if err != nil {
		return nil, err
	}
	err = s.servicePacksController.OnUpdateProject(*oldProject, *project)
	if err != nil {
		return nil, err
	}
	err = s.ProjectService.Update(iamProject)
	if err != nil {
		return nil, err
	}
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
	result := make([]*model.Project, len(iamProjects))
	for i := range iamProjects {
		result[i], err = makeProject(iamProjects[i])
		if err != nil {
			return nil, err
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
	project.Extensions = nil
	return &model.Project{
		Project:      *project,
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

// makeIamProject actually update extensions with servicepack
func makeIamProject(project *model.Project) (*iam.Project, error) {
	if project == nil {
		return nil, consts.ErrNilPointer
	}
	iamProject := project.Project
	extensions := project.Extensions
	if extensions == nil {
		extensions = map[consts.ObjectOrigin]*iam.Extension{}
	}
	extensions[consts.OriginFlantFlow] = &iam.Extension{
		Origin:    consts.OriginFlantFlow,
		OwnerType: iam.ProjectType,
		OwnerUUID: project.UUID,
		Attributes: map[string]interface{}{
			"service_packs": project.ServicePacks,
		},
	}
	iamProject.Extensions = extensions
	return &iamProject, nil
}

func (s *ProjectService) GetByID(pid model.ProjectUUID) (*model.Project, error) {
	iamProject, err := s.ProjectService.GetByID(pid)
	if err != nil {
		return nil, err
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
