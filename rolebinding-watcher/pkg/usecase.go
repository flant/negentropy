package pkg

import (
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type UserEffectiveRolesServiceImpl struct {
	db *io.MemoryStoreTxn
}

func UserEffectiveRolesService(db *io.MemoryStoreTxn) *UserEffectiveRolesServiceImpl {
	return &UserEffectiveRolesServiceImpl{db: db}
}

// Update also create and delete
func (s *UserEffectiveRolesServiceImpl) Update(uer *UserEffectiveRoles) error {
	repo := NewUserEffectiveRolesRepository(s.db)
	if len(uer.Tenants) == 0 {
		return s.delete(uer.Key())
	}
	return repo.Save(uer)
}

// Delete item if not exist - doesn,t provide error
func (s *UserEffectiveRolesServiceImpl) delete(key UserEffectiveRolesKey) error {
	repo := NewUserEffectiveRolesRepository(s.db)
	stored, err := repo.GetByKey(key)
	if err != nil {
		return err
	}
	if stored == nil {
		return nil
	}
	return repo.Delete(stored)
}

// GetByKey returns UserEffectiveRoles, if not exists return empty item
func (s *UserEffectiveRolesServiceImpl) GetByKey(key UserEffectiveRolesKey) (*UserEffectiveRoles, error) {
	userEffectiveRoles, err := NewUserEffectiveRolesRepository(s.db).GetByKey(key)
	if err != nil {
		return nil, err
	}
	if userEffectiveRoles == nil {
		return &UserEffectiveRoles{
			UserUUID: key.UserUUID,
			RoleName: key.RoleName,
			Tenants:  nil,
		}, nil
	}
	return userEffectiveRoles, nil
}
