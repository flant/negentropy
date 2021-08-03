package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func IdentityShares(db *io.MemoryStoreTxn) *IdentitySharingService {
	return &IdentitySharingService{
		db:         db,
		sharesRepo: model.NewIdentitySharingRepository(db),
		tenantRepo: model.NewTenantRepository(db),
	}
}

type IdentitySharingService struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics

	sharesRepo *model.IdentitySharingRepository
	tenantRepo *model.TenantRepository
}

func (s *IdentitySharingService) GetByID(id model.IdentitySharingUUID) (*model.IdentitySharing, error) {
	return s.sharesRepo.GetByID(id)
}

func (s *IdentitySharingService) Create(is *model.IdentitySharing) error {
	_, err := s.tenantRepo.GetByID(is.SourceTenantUUID)
	if err != nil {
		return err
	}
	_, err = s.tenantRepo.GetByID(is.DestinationTenantUUID)
	if err != nil {
		return err
	}

	is.Version = model.NewResourceVersion()
	return s.sharesRepo.Create(is)
}

func (s *IdentitySharingService) Update(is *model.IdentitySharing) error {
	is.Version = model.NewResourceVersion()

	// Update
	return s.sharesRepo.Update(is)
}

func (s *IdentitySharingService) Delete(id model.IdentitySharingUUID) error {
	archivingTimestamp, archivingHash := ArchivingLabel()
	return s.sharesRepo.Delete(id, archivingTimestamp, archivingHash)
}

func (s *IdentitySharingService) List(tid model.TenantUUID, showArchived bool) ([]*model.IdentitySharing, error) {
	return s.sharesRepo.List(tid, showArchived)
}
