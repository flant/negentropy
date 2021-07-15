package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleBindingService struct {
	db *io.MemoryStoreTxn
}

func RoleBindings(tx *io.MemoryStoreTxn) *RoleBindingService {
	return &RoleBindingService{tx}
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
	subj, err := NewSubjectsFetcher(s.db).Fetch(rb.Subjects)
	if err != nil {
		return err
	}
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users

	return model.NewRoleBindingRepository(s.db).Create(rb)
}

func (s *RoleBindingService) Update(rb *model.RoleBinding) error {
	// Validate
	if rb.Origin == "" {
		return model.ErrBadOrigin
	}

	// Validate tenant relation
	repo := model.NewRoleBindingRepository(s.db)
	stored, err := repo.GetByID(rb.UUID)
	if err != nil {
		return err
	}
	if stored.TenantUUID != rb.TenantUUID {
		return model.ErrNotFound
	}

	// Refill data
	subj, err := NewSubjectsFetcher(s.db).Fetch(rb.Subjects)
	if err != nil {
		return err
	}
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users

	// Preserve fields, that are not always accessable from the outside, e.g. from HTTP API
	if rb.Extensions == nil {
		rb.Extensions = stored.Extensions
	}

	// Store
	return model.NewRoleBindingRepository(s.db).Update(rb)
}

func (s *RoleBindingService) Delete(origin model.ObjectOrigin, id model.RoleBindingUUID) error {
	repo := model.NewRoleBindingRepository(s.db)
	roleBinding, err := repo.GetByID(id)
	if err != nil {
		return err
	}
	if roleBinding.Origin != origin {
		return model.ErrBadOrigin
	}
	return repo.Delete(id)
}

func (s *RoleBindingService) DeleteByTenant(tid model.TenantUUID) error {
	_, err := s.db.DeleteAll(model.RoleBindingType, model.TenantForeignPK, tid)
	return err
}

func (s *RoleBindingService) SetExtension(ext *model.Extension) error {
	repo := model.NewRoleBindingRepository(s.db)
	obj, err := repo.GetByID(ext.OwnerUUID)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		obj.Extensions = make(map[model.ObjectOrigin]*model.Extension)
	}
	obj.Extensions[ext.Origin] = ext
	return repo.Update(obj)
}

func (s *RoleBindingService) UnsetExtension(origin model.ObjectOrigin, rbid model.RoleBindingUUID) error {
	repo := model.NewRoleBindingRepository(s.db)
	obj, err := repo.GetByID(rbid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return s.Update(obj)
}
