package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type UserService struct {
	tenantUUID model.TenantUUID

	tenantRepo *iam_repo.TenantRepository
	usersRepo  *iam_repo.UserRepository
}

func Users(db *io.MemoryStoreTxn, tenantUUID model.TenantUUID) *UserService {
	return &UserService{
		tenantUUID: tenantUUID,

		tenantRepo: iam_repo.NewTenantRepository(db),
		usersRepo:  iam_repo.NewUserRepository(db),
	}
}

func (s *UserService) Create(user *model.User) error {
	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	user.Version = iam_repo.NewResourceVersion()
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier
	if user.Origin == "" {
		return consts.ErrBadOrigin
	}
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

	// Validate
	if stored.TenantUUID != s.tenantUUID {
		return consts.ErrNotFound
	}
	if stored.Version != user.Version {
		return consts.ErrBadVersion
	}
	if stored.Origin != user.Origin {
		return consts.ErrBadOrigin
	}

	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	// Update
	user.TenantUUID = s.tenantUUID
	user.Version = iam_repo.NewResourceVersion()
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier

	// Preserve fields, that are not always accessible from the outside, e.g. from HTTP API
	if user.Extensions == nil {
		user.Extensions = stored.Extensions
	}
	return s.usersRepo.Update(user)
}

func (s *UserService) Delete(origin model.ObjectOrigin, id model.UserUUID) error {
	user, err := s.usersRepo.GetByID(id)
	if err != nil {
		return err
	}
	if user.Origin != origin {
		return consts.ErrBadOrigin
	}

	err = s.usersRepo.CleanChildrenSliceIndexes(id)
	if err != nil {
		return err
	}
	archivingTimestamp, archivingHash := ArchivingLabel()
	return s.usersRepo.CascadeDelete(id, archivingTimestamp, archivingHash)
}

func (s *UserService) SetExtension(ext *model.Extension) error {
	obj, err := s.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[model.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return s.Update(obj)
}

func (s *UserService) UnsetExtension(origin model.ObjectOrigin, uuid model.UserUUID) error {
	obj, err := s.GetByID(uuid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return s.Update(obj)
}

func (s *UserService) Restore(id model.UserUUID) (*model.User, error) {
	return s.usersRepo.CascadeRestore(id)
}
