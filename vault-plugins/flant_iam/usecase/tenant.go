package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type TenantService struct {
	repo *model.TenantRepository

	// subtenants
	subTenantDeleters []DeleterByParent
}

func Tenants(db *io.MemoryStoreTxn) *TenantService {
	return &TenantService{
		repo: model.NewTenantRepository(db),
		subTenantDeleters: []DeleterByParent{
			UserDeleter(db),
		},
	}
}

func (s *TenantService) Create(t *model.Tenant) error {
	t.Version = model.NewResourceVersion()
	return s.repo.Create(t)
}

func (s *TenantService) Update(updated *model.Tenant) error {
	stored, err := s.repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}

	// Validate

	if stored.Version != updated.Version {
		return model.ErrBadVersion
	}
	updated.Version = model.NewResourceVersion()

	// Update

	return s.repo.Create(updated)
}

func (s *TenantService) Delete(id model.TenantUUID) error {
	if err := deleteChildren(id, s.subTenantDeleters); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *TenantService) GetByID(id model.TenantUUID) (*model.Tenant, error) {
	return s.repo.GetByID(id)
}

func (s *TenantService) List() ([]*model.Tenant, error) {
	return s.repo.List()
}
