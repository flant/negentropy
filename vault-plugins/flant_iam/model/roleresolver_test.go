package model

import (
	"testing"
)

func Test_collectAllRolesAndRoleBindings(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	// test for private method
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
	// test for private method
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
	rr := NewRoleResolver(tx)

	hasRole, gotParams, err := rr.CheckUserForProjectScopedRole(userUUID1, roleName1, tenantUUID1, projectUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
	expectedParams := RoleBindingParams{ValidTill: 120, RequireMFA: false, Options: map[string]interface{}{"o1": "data3"}}
	checkDeepEqual(t, expectedParams, gotParams)
}

func Test_collectAllRoleBindingsForServiceAccount(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	// test for private method
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
	rr := NewRoleResolver(tx)

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
	rr := NewRoleResolver(tx)

	hasRole, gotParams, err := rr.CheckUserForTenantScopedRole(userUUID2, roleName9, tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
	expectedParams := RoleBindingParams{ValidTill: 190, RequireMFA: false, Options: map[string]interface{}{"o1": "data9"}}
	checkDeepEqual(t, expectedParams, gotParams)
}

func Test_CheckServiceAccountForTenantScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := NewRoleResolver(tx)

	hasRole, gotParams, err := rr.CheckServiceAccountForTenantScopedRole(serviceAccountUUID2, roleName9, tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
	expectedParams := RoleBindingParams{ValidTill: 180, RequireMFA: false, Options: map[string]interface{}{"o1": "data8"}}
	checkDeepEqual(t, expectedParams, gotParams)
}

func Test_FindSubjectsWithProjectScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := NewRoleResolver(tx)

	users, serviceAccounts, err := rr.FindSubjectsWithProjectScopedRole(roleName1, tenantUUID1, projectUUID3)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{userUUID1: {}, userUUID2: {}, userUUID3: {}, userUUID5: {}}, stringSet(users))
	checkDeepEqual(t, map[string]struct{}{serviceAccountUUID1: {}, serviceAccountUUID2: {}, serviceAccountUUID4: {}},
		stringSet(serviceAccounts))
}

func Test_FindSubjectsWithTenantScopedRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := NewRoleResolver(tx)

	users, serviceAccounts, err := rr.FindSubjectsWithTenantScopedRole(roleName9, tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, map[string]struct{}{userUUID1: {}, userUUID2: {}, userUUID3: {}}, stringSet(users))
	checkDeepEqual(t, map[string]struct{}{serviceAccountUUID2: {}, serviceAccountUUID3: {}},
		stringSet(serviceAccounts))
}

func Test_CheckGroupForRole(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := NewRoleResolver(tx)

	hasRole, err := rr.CheckGroupForRole(groupUUID2, roleName1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, hasRole)
}

func Test_CheckGroupForRoleNegative(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		roleBindingFixture).Txn(true)
	rr := NewRoleResolver(tx)

	hasRole, err := rr.CheckGroupForRole(groupUUID2, roleName4)

	dieOnErr(t, err)
	checkDeepEqual(t, false, hasRole)
}

func Test_IsUserSharedWithTenant(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		identitySharingFixture).Txn(true)
	rr := NewRoleResolver(tx)

	isShared, err := rr.IsUserSharedWithTenant(&user1, tenantUUID2)

	dieOnErr(t, err)
	checkDeepEqual(t, true, isShared)
}

func Test_IsServiceAccountSharedWithTenant(t *testing.T) {
	tx := runFixtures(t, tenantFixture, userFixture, serviceAccountFixture, groupFixture, roleFixture, projectFixture,
		identitySharingFixture).Txn(true)
	rr := NewRoleResolver(tx)

	isShared, err := rr.IsServiceAccountSharedWithTenant(&sa4, tenantUUID1)

	dieOnErr(t, err)
	checkDeepEqual(t, true, isShared)
}
