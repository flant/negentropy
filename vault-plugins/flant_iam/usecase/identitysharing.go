package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func IdentityShares(db *io.MemoryStoreTxn) *IdentitySharingService {
	return &IdentitySharingService{
		db:         db,
		sharesRepo: repo.NewIdentitySharingRepository(db),
		tenantRepo: repo.NewTenantRepository(db),
	}
}

type IdentitySharingService struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics

	sharesRepo *repo.IdentitySharingRepository
	tenantRepo *repo.TenantRepository
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

	is.Version = repo.NewResourceVersion()
	return s.sharesRepo.Create(is)
}

func (s *IdentitySharingService) Update(is *model.IdentitySharing) error {
	is.Version = repo.NewResourceVersion()

	// Update
	return s.sharesRepo.Update(is)
}

func (s *IdentitySharingService) Delete(id model.IdentitySharingUUID) error {
	return s.sharesRepo.Delete(id, memdb.NewArchiveMark())
}

func (s *IdentitySharingService) List(tid model.TenantUUID, showArchived bool) ([]*model.IdentitySharing, error) {
	return s.sharesRepo.List(tid, showArchived)
}
