package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleBindingService struct {
	repo        *model.RoleBindingRepository
	tenantsRepo *model.TenantRepository

	memberFetcher *MembersFetcher
}

func RoleBindings(db *io.MemoryStoreTxn) *RoleBindingService {
	return &RoleBindingService{
		repo:          model.NewRoleBindingRepository(db),
		memberFetcher: NewMembersFetcher(db),
		tenantsRepo:   model.NewTenantRepository(db),
	}
}

func (s *RoleBindingService) Create(rb *model.RoleBinding) error {
	// Validate
	if rb.Origin == "" {
		return model.ErrBadOrigin
	}
	if rb.Version != "" {
		return model.ErrBadVersion
	}
	rb.Version = model.NewResourceVersion()

	// Refill data
	subj, err := s.memberFetcher.Fetch(rb.Members)
	if err != nil {
		return err
	}
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users
	rb.UUID = uuid.New()

	return s.repo.Create(rb)
}

func (s *RoleBindingService) Update(rb *model.RoleBinding) error {
	// Validate
	if rb.Origin == "" {
		return model.ErrBadOrigin
	}

	// Validate tenant relation
	stored, err := s.repo.GetByID(rb.UUID)
	if err != nil {
		return err
	}
	if stored.TenantUUID != rb.TenantUUID {
		return model.ErrNotFound
	}

	// Refill data
	subj, err := s.memberFetcher.Fetch(rb.Members)
	if err != nil {
		return err
	}
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users

	// Preserve fields, that are not always accessible from the outside, e.g. from HTTP API
	if rb.Extensions == nil {
		rb.Extensions = stored.Extensions
	}
	rb.Identifier = stored.Identifier

	// Store
	return s.repo.Update(rb)
}

func (s *RoleBindingService) Delete(origin model.ObjectOrigin, id model.RoleBindingUUID,
	archivingTimestamp model.UnixTime, archivingHash int64) error {
	roleBinding, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if roleBinding.Origin != origin {
		return model.ErrBadOrigin
	}
	return s.repo.Delete(id, archivingTimestamp, archivingHash)
}

func (s *RoleBindingService) SetExtension(ext *model.Extension) error {
	obj, err := s.repo.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[model.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return s.repo.Update(obj)
}

func (s *RoleBindingService) UnsetExtension(origin model.ObjectOrigin, rbid model.RoleBindingUUID) error {
	obj, err := s.repo.GetByID(rbid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return s.repo.Update(obj)
}
