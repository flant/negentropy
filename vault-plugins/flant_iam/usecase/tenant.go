package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type TenantService struct {
	repo *iam_repo.TenantRepository

	// subtenants
	childrenDeleters []DeleterByParent
}

func Tenants(db *io.MemoryStoreTxn) *TenantService {
	return &TenantService{
		repo: iam_repo.NewTenantRepository(db),
		childrenDeleters: []DeleterByParent{
			NewIdentitySharingDeleter(db),
			UserDeleter(db),
			ServiceAccountDeleter(db),
			GroupDeleter(db),
			RoleBindingDeleter(db),
			ProjectDeleter(db),
		},
	}
}

func (s *TenantService) Create(t *model.Tenant) error {
	t.Version = iam_repo.NewResourceVersion()
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
	updated.Version = iam_repo.NewResourceVersion()

	// Update

	return s.repo.Create(updated)
}

func (s *TenantService) Delete(id model.TenantUUID) error {
	archivingTimestamp, archivingHash := ArchivingLabel()
	if err := deleteChildren(id, s.childrenDeleters, archivingTimestamp, archivingHash); err != nil {
		return err
	}
	return s.repo.Delete(id, archivingTimestamp, archivingHash)
}

func (s *TenantService) GetByID(id model.TenantUUID) (*model.Tenant, error) {
	return s.repo.GetByID(id)
}

func (s *TenantService) List(showArchived bool) ([]*model.Tenant, error) {
	return s.repo.List(showArchived)
}

func (s *TenantService) Restore(id model.TenantUUID, fullRestore bool) (*model.Tenant, error) {
	if fullRestore {
		// TODO check if full restore available
		// TODO FullRestore
		return s.repo.Restore(id)
	}
	// TODO Short Restore
	return s.repo.Restore(id)
}
