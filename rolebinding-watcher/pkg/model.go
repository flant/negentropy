package pkg

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
)

const UserEffectiveRolesType = "user_effective_roles" // also, memdb schema name

type UserEffectiveRolesKey struct {
	UserUUID UserUUID `json:"user_uuid"` // PK
	RoleName RoleName `json:"role_name"`
}

type UserEffectiveRoles struct {
	UserUUID UserUUID                          `json:"user_uuid"` // PK
	RoleName RoleName                          `json:"role_name"`
	Tenants  []authz.EffectiveRoleTenantResult `json:"tenants"`
}

func (u *UserEffectiveRoles) Key() UserEffectiveRolesKey {
	if u == nil {
		return UserEffectiveRolesKey{}
	}
	return UserEffectiveRolesKey{
		UserUUID: u.UserUUID,
		RoleName: u.RoleName,
	}
}

func (u *UserEffectiveRoles) ObjType() string {
	return UserEffectiveRolesType
}

func (u *UserEffectiveRoles) ObjId() string {
	return u.UserUUID + "@" + u.RoleName
}

func (u *UserEffectiveRoles) NotEqual(other *UserEffectiveRoles) bool {
	return !u.Equal(other)
}

func (u *UserEffectiveRoles) Equal(other *UserEffectiveRoles) bool {
	if u == nil || other == nil {
		return false
	}
	if u.UserUUID != other.UserUUID ||
		u.RoleName != other.RoleName ||
		len(u.Tenants) != len(other.Tenants) {
		return false
	}
	for i := range u.Tenants {
		if u.Tenants[i].NotEqual(other.Tenants[i]) {
			return false
		}
	}
	return true
}
