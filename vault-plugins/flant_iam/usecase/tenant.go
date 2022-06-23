package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type TenantService struct {
	repo   *iam_repo.TenantRepository
	Origin consts.ObjectOrigin
}

func Tenants(db *io.MemoryStoreTxn, origin consts.ObjectOrigin) *TenantService {
	return &TenantService{
		repo:   iam_repo.NewTenantRepository(db),
		Origin: origin,
	}
}

func (s *TenantService) Create(tenant *model.Tenant) error {
	tenant.Version = iam_repo.NewResourceVersion()
	tenant.Origin = s.Origin
	return s.repo.Create(tenant)
}

func (s *TenantService) Update(updated *model.Tenant) error {
	stored, err := s.repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}
	if stored.Origin != s.Origin {
		return consts.ErrBadOrigin
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}

	// Validate

	if stored.Version != updated.Version {
		return consts.ErrBadVersion
	}
	updated.Version = iam_repo.NewResourceVersion()
	updated.Origin = s.Origin
	// Update

	return s.repo.Create(updated)
}

func (s *TenantService) Delete(id model.TenantUUID) error {
	stored, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if stored.Origin != s.Origin {
		return consts.ErrBadOrigin
	}

	return s.repo.CascadeDelete(id, memdb.NewArchiveMark())
}

func (s *TenantService) GetByID(id model.TenantUUID) (*model.Tenant, error) {
	return s.repo.GetByID(id)
}

func (s *TenantService) List(showArchived bool) ([]*model.Tenant, error) {
	return s.repo.List(showArchived)
}

func (s *TenantService) Restore(id model.TenantUUID, fullRestore bool) (*model.Tenant, error) {
	stored, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if stored.Origin != s.Origin {
		return nil, consts.ErrBadOrigin
	}
	if fullRestore {
		// TODO check if full restore available
		return s.repo.CascadeRestore(id)
	}
	return s.repo.Restore(id)
}
