package model

import (
	"testing"
)

func Test_collectAllRolesAndRoleBindings(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		ri:  NewRoleRepository(tx),
		gi:  NewGroupRepository(tx),
		rbi: NewRoleBindingRepository(tx),
	}

	roles, roleBindings, err := rr.collectAllRolesAndRoleBindings(tenantUUID1, roleName1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{roleName1: {}, roleName3: {}, roleName4: {}, roleName5: {}}, roles)
	checkDeepEqual(t, map[string]struct{}{
		rbUUID1: {},
		rbUUID3: {},
		rbUUID5: {},
	}, roleBindingsUUIDsFromMap(roleBindings))
}

func Test_collectAllRoleBindingsForUser(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		ri:  NewRoleRepository(tx),
		gi:  NewGroupRepository(tx),
		rbi: NewRoleBindingRepository(tx),
	}

	roleBindings, err := rr.collectAllRoleBindingsForUser(tenantUUID1, userUUID1)

	dieOnErr(t, err)
	rbUUIDS := roleBindingsUUIDsFromMap(roleBindings)
	checkDeepEqual(t, map[string]struct{}{rbUUID1: {}, rbUUID3: {}, rbUUID4: {}, rbUUID7: {}}, rbUUIDS)
}

func Test_CheckUserForProjectScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		ri:  NewRoleRepository(tx),
		gi:  NewGroupRepository(tx),
		rbi: NewRoleBindingRepository(tx),
	}

	hasRole, gotParams, err := rr.CheckUserForProjectScopedRole(userUUID1, roleName1, tenantUUID1, projectUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
	expectedParams := RoleBindingParams{ValidTill: 120, RequireMFA: false, Options: map[string]interface{}{"o1": "data3"}}
	checkDeepEqual(t, expectedParams, gotParams)
}

func Test_collectAllRoleBindingsForServiceAccount(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		ri:  NewRoleRepository(tx),
		gi:  NewGroupRepository(tx),
		rbi: NewRoleBindingRepository(tx),
	}

	roleBindings, err := rr.collectAllRoleBindingsForServiceAccount(tenantUUID1, serviceAccountUUID1)

	dieOnErr(t, err)
	rbUUIDS := roleBindingsUUIDsFromMap(roleBindings)
	checkDeepEqual(t, map[string]struct{}{
		rbUUID1: {},
		rbUUID3: {},
		rbUUID5: {},
	}, rbUUIDS)
}

func Test_CheckServiceAccountForProjectScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		ri:  NewRoleRepository(tx),
		gi:  NewGroupRepository(tx),
		rbi: NewRoleBindingRepository(tx),
	}

	hasRole, gotParams, err := rr.CheckServiceAccountForProjectScopedRole(serviceAccountUUID1, roleName1,
		tenantUUID1, projectUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
	expectedParams := RoleBindingParams{ValidTill: 160, RequireMFA: false, Options: map[string]interface{}{"o1": "data6"}}
	checkDeepEqual(t, expectedParams, gotParams)
}

func Test_CheckUserForTenantScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		ri:  NewRoleRepository(tx),
		gi:  NewGroupRepository(tx),
		rbi: NewRoleBindingRepository(tx),
	}

	hasRole, gotParams, err := rr.CheckUserForTenantScopedRole(userUUID2, roleName9, tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
	expectedParams := RoleBindingParams{ValidTill: 190, RequireMFA: false, Options: map[string]interface{}{"o1": "data9"}}
	checkDeepEqual(t, expectedParams, gotParams)
}

func Test_CheckServiceAccountForTenantScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := roleResolver{
		ri:  NewRoleRepository(tx),
		gi:  NewGroupRepository(tx),
		rbi: NewRoleBindingRepository(tx),
	}

	hasRole, gotParams, err := rr.CheckServiceAccountForTenantScopedRole(serviceAccountUUID2, roleName9, tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
	expectedParams := RoleBindingParams{ValidTill: 180, RequireMFA: false, Options: map[string]interface{}{"o1": "data8"}}
	checkDeepEqual(t, expectedParams, gotParams)
}
