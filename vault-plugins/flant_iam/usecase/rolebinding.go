package usecase

import (
	"errors"
	"fmt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type RoleBindingService struct {
	repo        *iam_repo.RoleBindingRepository
	tenantsRepo *iam_repo.TenantRepository

	memberFetcher *MembersFetcher
}

func RoleBindings(db *io.MemoryStoreTxn) *RoleBindingService {
	return &RoleBindingService{
		repo:          iam_repo.NewRoleBindingRepository(db),
		memberFetcher: NewMembersFetcher(db),
		tenantsRepo:   iam_repo.NewTenantRepository(db),
	}
}

func (s *RoleBindingService) Create(rb *model.RoleBinding) error {
	_, err := s.repo.GetByIdentifierAtTenant(rb.TenantUUID, rb.Identifier)
	if err != nil && !errors.Is(err, consts.ErrNotFound) {
		return err
	}
	if err == nil {
		return fmt.Errorf("%w: identifier:%s at tenant:%s", consts.ErrAlreadyExists, rb.Identifier, rb.TenantUUID)
	}
	// Validate
	if rb.Origin == "" {
		return consts.ErrBadOrigin
	}
	if rb.Version != "" {
		return consts.ErrBadVersion
	}
	rb.Version = iam_repo.NewResourceVersion()

	// Refill data
	subj, err := s.memberFetcher.Fetch(rb.Members)
	if err != nil {
		return fmt.Errorf("RoleBindingService.Create:%s", err)
	}
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users
	if rb.UUID == "" {
		rb.UUID = uuid.New()
	}

	return s.repo.Create(rb)
}

func (s *RoleBindingService) Update(rb *model.RoleBinding) error {
	// Validate
	stored, err := s.repo.GetByID(rb.UUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	if rb.Origin != stored.Origin {
		return consts.ErrBadOrigin
	}
	if stored.TenantUUID != rb.TenantUUID {
		return consts.ErrNotFound
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

func (s *RoleBindingService) Delete(origin consts.ObjectOrigin, id model.RoleBindingUUID) error {
	roleBinding, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if roleBinding.Origin != origin {
		return consts.ErrBadOrigin
	}
	return s.repo.CascadeDelete(id, memdb.NewArchiveMark())
}

func (s *RoleBindingService) SetExtension(ext *model.Extension) error {
	obj, err := s.repo.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Archived() {
		return consts.ErrIsArchived
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[consts.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return s.repo.Update(obj)
}

func (s *RoleBindingService) UnsetExtension(origin consts.ObjectOrigin, id model.RoleBindingUUID) error {
	obj, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if obj.Archived() {
		return consts.ErrIsArchived
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return s.repo.Update(obj)
}

func (s *RoleBindingService) List(tid model.TenantUUID, showArchived bool) ([]*model.RoleBinding, error) {
	return s.repo.List(tid, showArchived)
}

func (s *RoleBindingService) GetByID(id model.RoleBindingUUID) (*model.RoleBinding, error) {
	return s.repo.GetByID(id)
}
