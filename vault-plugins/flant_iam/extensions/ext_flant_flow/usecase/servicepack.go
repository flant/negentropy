package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ServicePackService struct {
	repo *repo.ServicePackRepository
}

func ServicePacks(db *io.MemoryStoreTxn) *ServicePackService {
	return &ServicePackService{
		repo: repo.NewServicePackRepository(db),
	}
}

func (s *ServicePackService) GetByProject(projectUUID iam_model.ProjectUUID) ([]*model.ServicePack, error) {
	return s.repo.List(projectUUID, false)
}
