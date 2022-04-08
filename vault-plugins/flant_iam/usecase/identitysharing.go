package usecase

import (
	"fmt"

	"github.com/hashicorp/go-multierror"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func IdentityShares(db *io.MemoryStoreTxn, origin consts.ObjectOrigin) *IdentitySharingService {
	return &IdentitySharingService{
		db:         db,
		origin:     origin,
		sharesRepo: repo.NewIdentitySharingRepository(db),
		tenantRepo: repo.NewTenantRepository(db),
	}
}

type IdentitySharingService struct {
	db         *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	origin     consts.ObjectOrigin
	sharesRepo *repo.IdentitySharingRepository
	tenantRepo *repo.TenantRepository
}

func (s *IdentitySharingService) GetByID(id model.IdentitySharingUUID) (*model.IdentitySharing, error) {
	return s.sharesRepo.GetByID(id)
}

func (s *IdentitySharingService) Create(is *model.IdentitySharing) error {
	is.Origin = s.origin
	_, err := s.tenantRepo.GetByID(is.SourceTenantUUID)
	if err != nil {
		return err
	}
	_, err = s.tenantRepo.GetByID(is.DestinationTenantUUID)
	if err != nil {
		return err
	}
	err = checkMembersOwnedToTenant(s.db, model.Members{Groups: is.Groups}, is.SourceTenantUUID)
	if err != nil {
		return err
	}
	is.Version = repo.NewResourceVersion()
	return s.sharesRepo.Create(is)
}

func (s *IdentitySharingService) Update(is *model.IdentitySharing) error {
	is.Version = repo.NewResourceVersion()
	if is.Origin != s.origin {
		return consts.ErrBadOrigin
	}
	err := checkMembersOwnedToTenant(s.db, model.Members{Groups: is.Groups}, is.SourceTenantUUID)
	if err != nil {
		return err
	}
	return s.sharesRepo.Update(is)
}

func (s *IdentitySharingService) Delete(id model.IdentitySharingUUID) error {
	stored, err := s.sharesRepo.GetByID(id)
	if err != nil {
		return err
	}
	if stored.Origin != s.origin {
		return consts.ErrBadOrigin
	}
	return s.sharesRepo.Delete(id, memdb.NewArchiveMark())
}

func (s *IdentitySharingService) List(tid model.TenantUUID, showArchived bool) ([]*model.IdentitySharing, error) {
	return s.sharesRepo.List(tid, showArchived)
}

func checkMembersOwnedToTenant(db *io.MemoryStoreTxn, members model.Members, tenantUUID model.TenantUUID) error {
	var allErrs *multierror.Error
	// check groups
	groupRepo := repo.NewGroupRepository(db)
	for _, gUUID := range members.Groups {
		group, err := groupRepo.GetByID(gUUID)
		if err != nil {
			allErrs = multierror.Append(allErrs, fmt.Errorf("checking group %s:%w", gUUID, err))
		}
		if group.TenantUUID != tenantUUID {
			allErrs = multierror.Append(allErrs, fmt.Errorf("checking group %s:group from %s tenant", gUUID, group.TenantUUID))
		}
	}
	// check users
	userRepo := repo.NewUserRepository(db)
	for _, gUUID := range members.Users {
		user, err := userRepo.GetByID(gUUID)
		if err != nil {
			allErrs = multierror.Append(allErrs, fmt.Errorf("checking user %s:%w", gUUID, err))
		}
		if user.TenantUUID != tenantUUID {
			allErrs = multierror.Append(allErrs, fmt.Errorf("checking user %s:user from %s tenant", gUUID, user.TenantUUID))
		}
	}
	// check service_account
	serviceAccountRepo := repo.NewServiceAccountRepository(db)
	for _, gUUID := range members.ServiceAccounts {
		serviceAccount, err := serviceAccountRepo.GetByID(gUUID)
		if err != nil {
			allErrs = multierror.Append(allErrs, fmt.Errorf("checking service_account %s:%w", gUUID, err))
		}
		if serviceAccount.TenantUUID != tenantUUID {
			allErrs = multierror.Append(allErrs, fmt.Errorf("checking service_account %s:service_account from %s tenant", gUUID, serviceAccount.TenantUUID))
		}
	}
	if allErrs == nil {
		return nil
	}
	return allErrs
}
