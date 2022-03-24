package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type UserService struct {
	tenantUUID model.TenantUUID

	tenantRepo *iam_repo.TenantRepository
	usersRepo  *iam_repo.UserRepository

	origin consts.ObjectOrigin
}

func Users(db *io.MemoryStoreTxn, tenantUUID model.TenantUUID, origin consts.ObjectOrigin) *UserService {
	return &UserService{
		tenantUUID: tenantUUID,

		tenantRepo: iam_repo.NewTenantRepository(db),
		usersRepo:  iam_repo.NewUserRepository(db),

		origin: origin,
	}
}

func (s *UserService) Create(user *model.User) error {
	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	user.Version = iam_repo.NewResourceVersion()
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier
	user.Origin = s.origin

	return s.usersRepo.Create(user)
}

func (s *UserService) GetByID(id model.UserUUID) (*model.User, error) {
	return s.usersRepo.GetByID(id)
}

func (s *UserService) List(showArchived bool) ([]*model.User, error) {
	return s.usersRepo.List(s.tenantUUID, showArchived)
}

func (s *UserService) Update(user *model.User) error {
	stored, err := s.usersRepo.GetByID(user.UUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	// Validate
	if stored.TenantUUID != s.tenantUUID {
		return consts.ErrNotFound
	}
	if stored.Version != user.Version {
		return consts.ErrBadVersion
	}
	if stored.Origin != s.origin {
		return consts.ErrBadOrigin
	}

	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	// Update
	user.TenantUUID = s.tenantUUID
	user.Version = iam_repo.NewResourceVersion()
	user.Origin = s.origin
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier

	// Preserve fields, that are not always accessible from the outside, e.g. from HTTP API
	if user.Extensions == nil {
		user.Extensions = stored.Extensions
	}
	return s.usersRepo.Update(user)
}

func (s *UserService) Delete(id model.UserUUID) error {
	user, err := s.usersRepo.GetByID(id)
	if err != nil {
		return err
	}
	if user.Origin != s.origin {
		return consts.ErrBadOrigin
	}

	err = s.usersRepo.CleanChildrenSliceIndexes(id)
	if err != nil {
		return err
	}
	return s.usersRepo.CascadeDelete(id, memdb.NewArchiveMark())
}

func (s *UserService) SetExtension(ext *model.Extension) error {
	stored, err := s.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	if stored.Extensions == nil {
		stored.Extensions = make(map[consts.ObjectOrigin]*model.Extension)
	}
	stored.Extensions[ext.Origin] = ext
	return s.Update(stored)
}

func (s *UserService) UnsetExtension(origin consts.ObjectOrigin, uuid model.UserUUID) error {
	stored, err := s.GetByID(uuid)
	if err != nil {
		return err
	}
	if stored.Archived() {
		return consts.ErrIsArchived
	}
	if stored.Extensions == nil {
		return nil
	}
	delete(stored.Extensions, origin)
	return s.Update(stored)
}

func (s *UserService) Restore(id model.UserUUID) (*model.User, error) {
	return s.usersRepo.CascadeRestore(id)
}
