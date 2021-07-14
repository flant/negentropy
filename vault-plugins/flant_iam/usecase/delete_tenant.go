package usecase

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type TenantDeleter struct {
	tx         *io.MemoryStoreTxn
	subdeleter Deleter
}

func NewTenantDeleter(tx *io.MemoryStoreTxn) *TenantDeleter {
	childDeleter := deleterSequence(
		&usersByTenantDeleter{tx},
		&serviceAccountsByTenantDeleter{tx},
		&groupsByTenantDeleter{tx},
		&roleBindingsByTenantDeleter{tx},
		&projectsByTenantDeleter{tx},
		&identitySharingsByTenantDeleter{tx},
		&serversByTenantDeleter{tx},
	)
	return &TenantDeleter{
		tx:         tx,
		subdeleter: childDeleter,
	}
}

func (d *TenantDeleter) Delete(tid model.TenantUUID) error {
	err := d.subdeleter.Delete(tid)
	if err != nil {
		return err
	}
	return model.NewTenantRepository(d.tx).Delete(tid)
}

type usersByTenantDeleter struct {
	tx *io.MemoryStoreTxn
}

func (d *usersByTenantDeleter) Delete(tid model.TenantUUID) error {
	users, err := model.NewUserRepository(d.tx).List(tid)
	if err != nil {
		return err
	}
	// remove each user and its child objects
	for _, user := range users {
		if err := NewUserDeleter(d.tx).Delete(user.UUID); err != nil {
			return err
		}
	}
	return nil
}

type serviceAccountsByTenantDeleter struct {
	tx *io.MemoryStoreTxn
}

func (d *serviceAccountsByTenantDeleter) Delete(tid model.TenantUUID) error {
	sas, err := model.NewServiceAccountRepository(d.tx).List(tid)
	if err != nil {
		return err
	}
	// remove each user and its child objects
	for _, sa := range sas {
		if err := NewServiceAccountDeleter(d.tx).Delete(sa.UUID); err != nil {
			return err
		}
	}
	return nil
}

type groupsByTenantDeleter struct {
	tx *io.MemoryStoreTxn
}

func (d *groupsByTenantDeleter) Delete(tid model.TenantUUID) error {
	// TODO @shvgn implement
	return nil
}

type roleBindingsByTenantDeleter struct {
	tx *io.MemoryStoreTxn
}

func (d *roleBindingsByTenantDeleter) Delete(tid model.TenantUUID) error {
	// TODO @shvgn implement
	return nil
}

type projectsByTenantDeleter struct {
	tx *io.MemoryStoreTxn
}

func (d *projectsByTenantDeleter) Delete(tid model.TenantUUID) error {
	// TODO @shvgn implement
	return nil
}

type identitySharingsByTenantDeleter struct {
	tx *io.MemoryStoreTxn
}

func (d *identitySharingsByTenantDeleter) Delete(tid model.TenantUUID) error {
	// TODO @shvgn implement
	return nil
}

type serversByTenantDeleter struct {
	tx *io.MemoryStoreTxn
}

func (d *serversByTenantDeleter) Delete(tid model.TenantUUID) error {
	// TODO @shvgn implement
	return nil
}
