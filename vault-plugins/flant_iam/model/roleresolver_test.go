package model

import (
	"testing"
)

func Test_CheckUserForProjectScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture, roleBindingFixture).Txn(true)
	repoRoles := NewRoleRepository(tx)
	repoGroups := NewGroupRepository(tx)
	repoRoleBindings := NewRoleBindingRepository(tx)
	rr := roleResolver{
		ri:  repoRoles,
		gi:  repoGroups,
		rbi: repoRoleBindings,
	}

	hasRole, gotParams, err := rr.CheckUserForProjectScopedRole(userUUID1, roleName1, tenantUUID1, projectUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
	expectedParams := RoleBindingParams{ValidTill: 200, RequireMFA: false, Options: map[string]interface{}{"o1": "data3"}}
	checkDeepEqual(t, expectedParams, gotParams)
}
