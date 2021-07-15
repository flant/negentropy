package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type TenantService struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func Tenants(db *io.MemoryStoreTxn) *TenantService {
	return &TenantService{db: db}
}

func (s *TenantService) Create(t *model.Tenant) error {
	t.Version = model.NewResourceVersion()
	return model.NewTenantRepository(s.db).Create(t)
}

func (s *TenantService) Update(updated *model.Tenant) error {
	repo := model.NewTenantRepository(s.db)

	stored, err := repo.GetByID(updated.UUID)
	if err != nil {
		return err
	}

	// Validate

	if stored.Version != updated.Version {
		return model.ErrBadVersion
	}
	updated.Version = model.NewResourceVersion()

	// Update

	return repo.Create(updated)
}

func (s *TenantService) Delete(id model.TenantUUID) error {
	return model.NewTenantRepository(s.db).Delete(id)
}
