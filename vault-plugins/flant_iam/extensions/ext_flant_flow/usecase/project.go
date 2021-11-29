package usecase

import (
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ProjectService struct {
	*iam_usecase.ProjectService
}

func Projects(db *io.MemoryStoreTxn) *ProjectService {
	return &ProjectService{
		ProjectService: iam_usecase.Projects(db, consts.OriginFlantFlow),
	}
}

func (s *ProjectService) Create(project *model.Project) error {
	// TODO verify servicepacks
	// TODO fix servicepacks params due to default teams
	iamProject, err := makeIamProject(project)
	if err != nil {
		return err
	}
	return s.ProjectService.Create(iamProject)
}

func (s *ProjectService) Update(project *model.Project) error {
	stored, err := s.ProjectService.GetByID(project.UUID)
	if err != nil {
		return err
	}
	project.Extensions = stored.Extensions
	iamProject, err := makeIamProject(project)
	if err != nil {
		return err
	}
	return s.ProjectService.Update(iamProject)
}

// func (s *ProjectService) Delete(id model.ProjectUUID) error {}

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
	var servicePacks map[model.ServicePackName]string
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

func unmarshallServicePackCandidate(servicePacksRaw interface{}) (map[model.ServicePackName]string, error) {
	var servicePacks map[model.ServicePackName]string
	switch spMap := servicePacksRaw.(type) {
	// after kafka restoration
	case map[model.ServicePackName]interface{}:
		servicePacks = map[model.ServicePackName]string{}
		for k, v := range spMap {
			value, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("%w:need string, passed:%T",
					consts.ErrWrongType, servicePacksRaw)
			}
			servicePacks[k] = value
		}
	case map[model.ServicePackName]string:
		servicePacks = spMap
	default:
		return nil, fmt.Errorf("%w:need map[string]interface{} or map[string]string, passed:%T",
			consts.ErrWrongType, servicePacksRaw)
	}
	return servicePacks, nil
}

// makeIamProject actually update extesions with servicepack
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
