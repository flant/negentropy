package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type RoleBindings struct {
	db *io.MemoryStoreTxn
}

func NewRoleBindings(tx *io.MemoryStoreTxn) *RoleBindings {
	return &RoleBindings{tx}
}

func (r *RoleBindings) Create(rb *model.RoleBinding) error {
	// Validate
	if rb.Origin == "" {
		return model.ErrBadOrigin
	}
	if rb.Version != "" {
		return model.ErrBadVersion
	}
	rb.Version = model.NewResourceVersion()

	// Refill data
	subj, err := NewSubjectsFetcher(r.db).Fetch(rb.Subjects)
	if err != nil {
		return err
	}
	rb.Groups = subj.Groups
	rb.ServiceAccounts = subj.ServiceAccounts
	rb.Users = subj.Users

	return model.NewRoleBindingRepository(r.db).Create(rb)
}

func (r *RoleBindings) Update(rb *model.RoleBinding) error {
	// Validate
	if rb.Origin == "" {
		return model.ErrBadOrigin
	}

	// Validate tenant relation
	repo := model.NewRoleBindingRepository(r.db)
	stored, err := repo.GetByID(rb.UUID)
	if err != nil {
		return err
	}
	if stored.TenantUUID != rb.TenantUUID {
		return model.ErrNotFound
	}

	// Refill data
	subj, err := NewSubjectsFetcher(r.db).Fetch(rb.Subjects)
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
	return model.NewRoleBindingRepository(r.db).Update(rb)
}

func (r *RoleBindings) Delete(origin model.ObjectOrigin, id model.RoleBindingUUID) error {
	repo := model.NewRoleBindingRepository(r.db)
	roleBinding, err := repo.GetByID(id)
	if err != nil {
		return err
	}
	if roleBinding.Origin != origin {
		return model.ErrBadOrigin
	}
	return repo.Delete(id)
}

func (r *RoleBindings) DeleteByTenant(tid model.TenantUUID) error {
	_, err := r.db.DeleteAll(model.RoleBindingType, model.TenantForeignPK, tid)
	return err
}

func (r *RoleBindings) SetExtension(ext *model.Extension) error {
	repo := model.NewRoleBindingRepository(r.db)
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

func (r *RoleBindings) UnsetExtension(origin model.ObjectOrigin, rbid model.RoleBindingUUID) error {
	repo := model.NewRoleBindingRepository(r.db)
	obj, err := repo.GetByID(rbid)
	if err != nil {
		return err
	}
	if obj.Extensions == nil {
		return nil
	}
	delete(obj.Extensions, origin)
	return r.Update(obj)
}
