package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type UserService struct {
	tenantUUID model.TenantUUID

	tenantRepo *model.TenantRepository
	usersRepo  *model.UserRepository

	childrenDeleters []DeleterByParent
}

func Users(db *io.MemoryStoreTxn, tenantUUID model.TenantUUID) *UserService {
	return &UserService{
		tenantUUID: tenantUUID,

		tenantRepo: model.NewTenantRepository(db),
		usersRepo:  model.NewUserRepository(db),

		childrenDeleters: []DeleterByParent{
			MultipassDeleter(db),
		},
	}
}

func (s *UserService) Create(user *model.User) error {
	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	user.Version = model.NewResourceVersion()
	user.FullIdentifier = user.Identifier + "@" + tenant.Identifier
	if user.Origin == "" {
		return model.ErrBadOrigin
	}
	return s.usersRepo.Create(user)
}

func (s *UserService) GetByID(id model.UserUUID) (*model.User, error) {
	return s.usersRepo.GetByID(id)
}

func (s *UserService) List() ([]*model.User, error) {
	return s.usersRepo.List(s.tenantUUID)
}

func (s *UserService) Update(user *model.User) error {
	stored, err := s.usersRepo.GetByID(user.UUID)
	if err != nil {
		return err
	}

	// Validate
	if stored.TenantUUID != s.tenantUUID {
		return model.ErrNotFound
	}
	if stored.Version != user.Version {
		return model.ErrBadVersion
	}
	if stored.Origin != user.Origin {
		return model.ErrBadOrigin
	}

	tenant, err := s.tenantRepo.GetByID(s.tenantUUID)
	if err != nil {
		return err
	}

	// Update
	user.TenantUUID = s.tenantUUID
	user.Version = model.NewResourceVersion()
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
		return model.ErrBadOrigin
	}

	if err := deleteChildren(id, s.childrenDeleters); err != nil {
		return err
	}

	return s.usersRepo.Delete(id)
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
