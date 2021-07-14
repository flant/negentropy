package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type UserWithOriginDeleter struct {
	UserDeleter

	origin model.ObjectOrigin
}

func (d *UserWithOriginDeleter) Delete(id model.UserUUID) error {
	user, err := model.NewUserRepository(d.tx).GetByID(id)
	if err != nil {
		return err
	}
	if user.Origin != d.origin {
		return model.ErrBadOrigin
	}
	return d.UserDeleter.Delete(id)
}

func NewUserWithOriginDeleter(tx *io.MemoryStoreTxn, origin model.ObjectOrigin) *UserWithOriginDeleter {
	return &UserWithOriginDeleter{
		origin: origin,
		UserDeleter: UserDeleter{
			tx: tx,
			subdeleter: deleterSequence(
				&multipassDeleter{tx: tx, ownerType: model.MultipassOwnerUser},
			),
		},
	}
}

type UserDeleter struct {
	tx         *io.MemoryStoreTxn
	subdeleter Deleter
}

func (d *UserDeleter) Delete(id model.UserUUID) error {
	err := d.subdeleter.Delete(id)
	if err != nil {
		return err
	}
	// TODO clean groups, rolebindings
	return model.NewUserRepository(d.tx).Delete(id)
}

func NewUserDeleter(tx *io.MemoryStoreTxn) *UserDeleter {
	return &UserDeleter{
		tx: tx,
		subdeleter: deleterSequence(
			&multipassDeleter{tx: tx, ownerType: model.MultipassOwnerUser},
		),
	}
}

type multipassDeleter struct {
	tx        *io.MemoryStoreTxn
	ownerType model.MultipassOwnerType
}

func (d *multipassDeleter) Delete(ownerID string) error {
	repo := model.NewMultipassRepository(d.tx)
	mps, err := repo.List(&model.Multipass{
		OwnerUUID: ownerID,
		OwnerType: d.ownerType,
	})
	if err != nil {
		return nil
	}
	for _, mp := range mps {
		if err := repo.Delete(&model.Multipass{UUID: mp.UUID}); err != nil {
			return err
		}
	}
	return nil
}
