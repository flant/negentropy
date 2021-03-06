package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type ServiceAccountPasswordsService struct {
	tenantUUID model.TenantUUID
	ownerUUID  model.ServiceAccountUUID

	repo       *iam_repo.ServiceAccountPasswordRepository
	saRepo     *iam_repo.ServiceAccountRepository
	tenantRepo *iam_repo.TenantRepository
}

func ServiceAccountPasswords(db *io.MemoryStoreTxn, tid model.TenantUUID, said model.ServiceAccountUUID) *ServiceAccountPasswordsService {
	return &ServiceAccountPasswordsService{
		tenantUUID: tid,
		ownerUUID:  said,

		repo:       iam_repo.NewServiceAccountPasswordRepository(db),
		saRepo:     iam_repo.NewServiceAccountRepository(db),
		tenantRepo: iam_repo.NewTenantRepository(db),
	}
}

func (r *ServiceAccountPasswordsService) Create(p *model.ServiceAccountPassword) error {
	err := r.validateContext()
	if err != nil {
		return err
	}
	return r.repo.Create(p)
}

func (r *ServiceAccountPasswordsService) Delete(id model.ServiceAccountPasswordUUID) error {
	err := r.validateContext()
	if err != nil {
		return err
	}
	return r.repo.Delete(id, memdb.NewArchiveMark())
}

func (r *ServiceAccountPasswordsService) GetByID(id model.ServiceAccountPasswordUUID) (*model.ServiceAccountPassword, error) {
	err := r.validateContext()
	if err != nil {
		return nil, err
	}
	return r.repo.GetByID(id)
}

func (r *ServiceAccountPasswordsService) List(showArchived bool) ([]*model.ServiceAccountPassword, error) {
	err := r.validateContext()
	if err != nil {
		return nil, err
	}
	return r.repo.List(r.ownerUUID, showArchived)
}

func (r *ServiceAccountPasswordsService) validateContext() error {
	if _, err := r.tenantRepo.GetByID(r.tenantUUID); err != nil {
		return err
	}

	owner, err := r.saRepo.GetByID(r.ownerUUID)
	if err != nil {
		return err
	}
	if owner.TenantUUID != r.tenantUUID {
		return consts.ErrNotFound
	}

	return nil
}
