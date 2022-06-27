package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/config"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
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
			newInternalProjectServicePackBuilder(db, liveConfig),
		},
	}
}
