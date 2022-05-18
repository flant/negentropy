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
		db:          db,
		origin:      origin,
		sharesRepo:  repo.NewIdentitySharingRepository(db),
		tenantRepo:  repo.NewTenantRepository(db),
		groupMapper: NewGroupMapper(db),
	}
}

type IdentitySharingService struct {
	db          *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
	origin      consts.ObjectOrigin
	sharesRepo  *repo.IdentitySharingRepository
	tenantRepo  *repo.TenantRepository
	groupMapper GroupMapper
}

func (s *IdentitySharingService) GetByID(id model.IdentitySharingUUID) (*DenormalizedIdentitySharing, error) {
	is, err := s.sharesRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return s.denormalizeIdentitySharing(is)
}

func (s *IdentitySharingService) Create(is *model.IdentitySharing) (*DenormalizedIdentitySharing, error) {
	is.Origin = s.origin
	_, err := s.tenantRepo.GetByID(is.SourceTenantUUID)
	if err != nil {
		return nil, err
	}
	_, err = s.tenantRepo.GetByID(is.DestinationTenantUUID)
	if err != nil {
		return nil, err
	}
	err = checkMembersOwnedToTenant(s.db, model.Members{Groups: is.Groups}, is.SourceTenantUUID)
	if err != nil {
		return nil, err
	}
	is.Version = repo.NewResourceVersion()
	err = s.sharesRepo.Create(is)
	if err != nil {
		return nil, err
	}
	return s.denormalizeIdentitySharing(is)
}

func (s *IdentitySharingService) Update(is *model.IdentitySharing) (*DenormalizedIdentitySharing, error) {
	stored, err := s.sharesRepo.GetByID(is.UUID)
	if err != nil {
		return nil, err
	}
	// Validate
	notAllowedChangeErr := fmt.Errorf("%w: allowed only change groups", consts.ErrInvalidArg)
	if is.SourceTenantUUID != stored.SourceTenantUUID {
		return nil, notAllowedChangeErr
	}

	if stored.Origin != s.origin {
		return nil, consts.ErrBadOrigin
	}
	if stored.Version != is.Version {
		return nil, consts.ErrBadVersion
	}
	if stored.Archived() {
		return nil, consts.ErrIsArchived
	}
	err = checkMembersOwnedToTenant(s.db, model.Members{Groups: is.Groups}, is.SourceTenantUUID)
	if err != nil {
		return nil, err
	}

	is.DestinationTenantUUID = stored.DestinationTenantUUID
	is.Origin = s.origin
	is.Version = repo.NewResourceVersion()
	err = s.sharesRepo.Update(is)
	if err != nil {
		return nil, err
	}
	return s.denormalizeIdentitySharing(is)
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

func (s *IdentitySharingService) List(tid model.TenantUUID, showArchived bool) ([]*DenormalizedIdentitySharing, error) {
	iss, err := s.sharesRepo.List(tid, showArchived)
	if err != nil {
		return nil, err
	}
	return s.denormalizeIdentitySharings(iss)
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

type DenormalizedIdentitySharing struct {
	memdb.ArchiveMark

	UUID                        model.IdentitySharingUUID `json:"uuid"` // PK
	SourceTenantUUID            model.TenantUUID          `json:"source_tenant_uuid"`
	DestinationTenantUUID       model.TenantUUID          `json:"destination_tenant_uuid"`
	DestinationTenantIdentifier model.TenantUUID          `json:"destination_tenant_identifier"`

	Version string `json:"resource_version"`

	Origin consts.ObjectOrigin `json:"origin"`

	// Groups which to share with target tenant
	Groups []GroupUUIDWithIdentifiers `json:"groups"`
}

func (s *IdentitySharingService) denormalizeIdentitySharings(iss []*model.IdentitySharing) ([]*DenormalizedIdentitySharing, error) {
	result := make([]*DenormalizedIdentitySharing, 0, len(iss))
	for _, is := range iss {
		dis, err := s.denormalizeIdentitySharing(is)
		if err != nil {
			return nil, err
		}
		result = append(result, dis)
	}
	return result, nil
}

func (s *IdentitySharingService) denormalizeIdentitySharing(is *model.IdentitySharing) (*DenormalizedIdentitySharing, error) {
	destinationTenant, err := s.tenantRepo.GetByID(is.DestinationTenantUUID)
	if err != nil {
		return nil, err
	}
	groupUUIDWithIdentifiers, err := s.groupMapper.Denormalize(is.Groups)
	if err != nil {
		return nil, err
	}
	return &DenormalizedIdentitySharing{
		ArchiveMark:                 is.ArchiveMark,
		UUID:                        is.UUID,
		SourceTenantUUID:            is.SourceTenantUUID,
		DestinationTenantUUID:       is.DestinationTenantUUID,
		DestinationTenantIdentifier: destinationTenant.Identifier,
		Version:                     is.Version,
		Origin:                      is.Origin,
		Groups:                      groupUUIDWithIdentifiers,
	}, nil
}
