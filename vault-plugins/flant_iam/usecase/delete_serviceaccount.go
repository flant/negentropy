package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ServiceAccountWithOriginDeleter struct {
	ServiceAccountDeleter

	origin model.ObjectOrigin
}

func (d *ServiceAccountWithOriginDeleter) Delete(id model.ServiceAccountUUID) error {
	sa, err := model.NewServiceAccountRepository(d.tx).GetByID(id)
	if err != nil {
		return err
	}
	if sa.Origin != d.origin {
		return model.ErrBadOrigin
	}
	return d.ServiceAccountDeleter.Delete(id)
}

func NewServiceAccountWithOriginDeleter(tx *io.MemoryStoreTxn, origin model.ObjectOrigin) *ServiceAccountWithOriginDeleter {
	return &ServiceAccountWithOriginDeleter{
		origin: origin,
		ServiceAccountDeleter: ServiceAccountDeleter{
			tx: tx,
			subdeleter: deleterSequence(
				&multipassDeleter{tx: tx, ownerType: model.MultipassOwnerServiceAccount},
			),
		},
	}
}

type ServiceAccountDeleter struct {
	tx         *io.MemoryStoreTxn
	subdeleter Deleter
}

func (d *ServiceAccountDeleter) Delete(id model.ServiceAccountUUID) error {
	err := d.subdeleter.Delete(id)
	if err != nil {
		return err
	}
	// TODO clean groups, rolebindings
	return model.NewServiceAccountRepository(d.tx).Delete(id)
}

func NewServiceAccountDeleter(tx *io.MemoryStoreTxn) *ServiceAccountDeleter {
	return &ServiceAccountDeleter{
		tx: tx,
		subdeleter: deleterSequence(
			&multipassDeleter{tx: tx, ownerType: model.MultipassOwnerServiceAccount},
			&serviceAccountPasswordDeleter{tx: tx},
		),
	}
}

type serviceAccountPasswordDeleter struct {
	tx *io.MemoryStoreTxn
}

func (d *serviceAccountPasswordDeleter) Delete(ownerID string) error {
	repo := model.NewServiceAccountPasswordRepository(d.tx)
	ps, err := repo.List(&model.ServiceAccountPassword{
		OwnerUUID: ownerID,
	})
	if err != nil {
		return nil
	}
	for _, p := range ps {
		if err := repo.Delete(&model.ServiceAccountPassword{UUID: p.UUID}); err != nil {
			return err
		}
	}
	return nil
}
