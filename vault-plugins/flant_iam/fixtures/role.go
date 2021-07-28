package fixtures

import (
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

const (
	RoleName1  = "RoleName1"
	RoleName2  = "RoleName2"
	RoleName3  = "RoleName3"
	RoleName4  = "RoleName4"
	RoleName5  = "RoleName5"
	RoleName6  = "RoleName6"
	RoleName7  = "RoleName7"
	RoleName8  = "RoleName8"
	RoleName9  = "RoleName9"
	RoleName10 = "RoleName10"
)

func Roles() []model.Role {
	return []model.Role{
		{
			Name:          RoleName1,
			Scope:         model.RoleScopeProject,
			IncludedRoles: nil,
		},
		{
			Name:          RoleName2,
			Scope:         model.RoleScopeProject,
			IncludedRoles: nil,
		},
		{
			Name:          RoleName3,
			Scope:         model.RoleScopeProject,
			IncludedRoles: []model.IncludedRole{{Name: RoleName1}},
		},
		{
			Name:          RoleName4,
			Scope:         model.RoleScopeProject,
			IncludedRoles: []model.IncludedRole{{Name: RoleName1}, {Name: RoleName2}},
		},
		{
			Name:          RoleName5,
			Scope:         model.RoleScopeProject,
			IncludedRoles: []model.IncludedRole{{Name: RoleName2}, {Name: RoleName3}},
		},
		{
			Name:  RoleName6,
			Scope: model.RoleScopeProject,
		},
		{
			Name:  RoleName7,
			Scope: model.RoleScopeProject,
		},
		{
			Name:  RoleName8,
			Scope: model.RoleScopeTenant,
		},
		{
			Name:  RoleName9,
			Scope: model.RoleScopeTenant,
		},
		{
			Name:          RoleName10,
			Scope:         model.RoleScopeTenant,
			IncludedRoles: []model.IncludedRole{{Name: RoleName9}},
		},
	}
}
