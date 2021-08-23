package txnwatchers

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

const (
	tenantUUID       = "00000000-0001-0000-0000-000000000000"
	userUUID1        = "00000000-0001-0000-0000-000000000000"
	userUUID2        = "00000000-0002-0000-0000-000000000000"
	userUUID3        = "00000000-0003-0000-0000-000000000000"
	userUUID4        = "00000000-0004-0000-0000-000000000000"
	userUUID5        = "00000000-0005-0000-0000-000000000000"
	groupUUID1       = "00000000-0001-0000-0000-000000000000"
	groupUUID2       = "00000000-0002-0000-0000-000000000000"
	groupUUID3       = "00000000-0003-0000-0000-000000000000"
	groupUUID4       = "00000000-0004-0000-0000-000000000000"
	groupUUID5       = "00000000-0005-0000-0000-000000000000"
	roleBindingUUID1 = "00000000-0001-0000-0000-000000000000"
	roleBindingUUID2 = "00000000-0002-0000-0000-000000000000"
	roleBindingUUID3 = "00000000-0003-0000-0000-000000000000"
)

func Test_FindAffectedUsersAndSAs_OnNewRoleBinding(t *testing.T) {
	var err error
	// Create db with some entities.
	mem := getTestMemoryStorage(t, fixtureForRoleBindingChange)

	// Should get all users when inserting new RoleBinding.
	rb := &model.RoleBinding{
		TenantUUID: tenantUUID,
		UUID:       userUUID1,
		Users:      []string{userUUID1},
		Groups:     []string{userUUID1, userUUID2, userUUID3}, // nested groups
		Roles: []model.BoundRole{
			{
				Name: "test",
			},
		},
	}

	// Open new transaction after Commit.
	txn := mem.Txn(true)
	users, sas, err := FindUsersAndSAsAffectedByPossibleRoleAddingOnRoleBindingChange(txn, nil, rb, "test")

	require.NoError(t, err)
	require.Len(t, users, 3, "Should get all users for new RoleBinding.")
	require.Len(t, sas, 0)
}

func Test_FindAffectedUsersAndSAs_OnRoleRemoveFromRoleBinding(t *testing.T) {
	var err error
	// Create db with some entities.
	mem := getTestMemoryStorage(t, fixtureForRoleBindingChange)

	// Should get all users when inserting new RoleBinding.
	rb := &model.RoleBinding{
		TenantUUID: tenantUUID,
		UUID:       userUUID1,
		Users:      []string{userUUID1},
		Groups:     []string{userUUID2}, // editors: userUUID2, userUUID3
		Roles: []model.BoundRole{
			{
				Name: "test",
			},
		},
	}

	// Should get no users when role is deleted from RoleBinding.
	newrb := &model.RoleBinding{
		TenantUUID: tenantUUID,
		UUID:       userUUID1,
		Groups:     []string{userUUID1}, // users
	}

	// Open new transaction after Commit.
	txn := mem.Txn(true)
	users, sas, err := FindUsersAndSAsAffectedByPossibleRoleAddingOnRoleBindingChange(txn, rb, newrb, "test")
	require.NoError(t, err)
	require.Len(t, users, 0, "Should get no users when role is deleted from RoleBinding.")
	require.Len(t, sas, 0)
}

func Test_FindAffectedUsersAndSAs_NoUsersForNonExistentRole(t *testing.T) {
	var err error
	// Create db with some entities.
	mem := getTestMemoryStorage(t, fixtureForRoleBindingChange)

	oldRoleBinding := &model.RoleBinding{
		TenantUUID: tenantUUID,
		UUID:       userUUID1,
		Users:      []string{userUUID1},
		Groups:     []string{userUUID2}, // editors: userUUID2, userUUID3
		Roles: []model.BoundRole{
			{
				Name: "test",
			},
		},
	}

	newRoleBinding := &model.RoleBinding{
		TenantUUID: tenantUUID,
		UUID:       userUUID1,
		Users:      []string{userUUID3},
		Roles: []model.BoundRole{
			{
				Name: "test",
			},
			{
				Name: "test2",
			},
		},
	}

	// Should return nil when role is not granted at all.
	txn := mem.Txn(true)
	users, sas, err := FindUsersAndSAsAffectedByPossibleRoleAddingOnRoleBindingChange(txn, oldRoleBinding, newRoleBinding, "non-existent-role")
	require.NoError(t, err)
	require.Nil(t, users, "Should not return affected users.")
	require.Nil(t, sas, "Should not return affected service accounts.")
}

func Test_FindAffectedUsersAndSAs_OnRoleBindingChange_ModifiedSubjects(t *testing.T) {
	var err error
	// Create db with some entities.
	mem := getTestMemoryStorage(t, fixtureForRoleBindingChange)

	oldRoleBinding := &model.RoleBinding{
		TenantUUID: tenantUUID,
		UUID:       userUUID1,
		Users:      []string{userUUID2, userUUID3},
		Roles: []model.BoundRole{
			{Name: "test"},
			{Name: "test2"},
		},
	}

	newRoleBinding := &model.RoleBinding{
		TenantUUID: tenantUUID,
		UUID:       userUUID1,
		Groups:     []string{userUUID3}, // admins: u1+u2
		Roles: []model.BoundRole{
			{
				Name: "test",
			},
		},
	}

	// Should return added users when role is not changed.
	txn := mem.Txn(true)
	users, sas, err := FindUsersAndSAsAffectedByPossibleRoleAddingOnRoleBindingChange(txn, oldRoleBinding, newRoleBinding, "test")
	require.NoError(t, err)
	require.Len(t, users, 2, "Should return 2 affected users.")
	require.Len(t, sas, 0, "Should not return affected service accounts.")
	require.Contains(t, users, userUUID1, "Should return user1 as affected.")
	require.Contains(t, users, userUUID2, "Should return user2 as affected.")
}

func Test_FindAffectedUsersAndSAs_OnGroupChange_ModifiedSubjects(t *testing.T) {
	var err error
	// Create db with some users and test role.
	mem := getTestMemoryStorage(t, fixtureForGroupChange)

	txn := mem.Txn(true)
	oldGroup := testGroup(groupUUID4, []string{userUUID4}, nil, []string{groupUUID2}) // u4 + u2

	createGroups(t, txn,
		testGroup(groupUUID3, []string{userUUID3}, nil, []string{groupUUID1}), // u3+u1
		oldGroup,
		// parent group for old group
		testGroup(groupUUID5, []string{userUUID5}, nil, []string{groupUUID4}),
	)

	// Add role binding for parent group with role "test".
	createRoleBindings(t, txn, &model.RoleBinding{
		TenantUUID: tenantUUID,
		UUID:       roleBindingUUID1,
		Groups:     []string{groupUUID5},
		Identifier: "test_role_binding1",
		Roles: []model.BoundRole{
			{
				Name: "test",
			},
		},
	})

	err = txn.Commit()
	require.NoError(t, err)

	newGroup := testGroup(groupUUID4, []string{userUUID4, userUUID5}, nil, []string{groupUUID2, groupUUID3}) // add u5 and u3+u1

	// Should return added users.
	txn = mem.Txn(true)
	users, sas, err := FindUsersAndSAsAffectedByPossibleRoleAddingOnGroupChange(txn, oldGroup, newGroup, "test")
	require.NoError(t, err)
	require.Len(t, users, 3, "Should return 3 affected users.")
	require.Len(t, sas, 0, "Should not return affected service accounts.")
	require.Contains(t, users, userUUID1, "Should return user1 as affected.")
	require.Contains(t, users, userUUID3, "Should return user3 as affected.")
	require.Contains(t, users, userUUID5, "Should return user5 as affected.")
}

// fixtureForRoleBindingChange
// 3 users: "user1", "user2", "user3"
// 3 groups: "users"(u1,u2,u3), "admins"(u1+editors), editors(u2),
// role "test"
func fixtureForRoleBindingChange(t *testing.T, txn *io.MemoryStoreTxn) {
	createUsers(t, txn,
		testUser(userUUID1),
		testUser(userUUID2),
		testUser(userUUID3),
	)

	createGroups(t, txn,
		testGroup(groupUUID1, []string{userUUID1, userUUID2, userUUID3}, nil, nil),
		testGroup(groupUUID2, []string{userUUID2}, nil, nil),
		testGroup(groupUUID3, []string{userUUID1}, nil, []string{groupUUID2}),
	)

	createRoles(t, txn,
		testRole("test", model.RoleScopeProject),
		testRole("test2", model.RoleScopeProject),
	)
}

// fixtureForGroupChange
// 3 users: "user1", "user2", "user3"
// 3 groups: "users"(u1,u2,u3), "admins"(u1+editors), editors(u2),
// role "test"
func fixtureForGroupChange(t *testing.T, txn *io.MemoryStoreTxn) {
	createUsers(t, txn,
		testUser(userUUID1),
		testUser(userUUID2),
		testUser(userUUID3),
		testUser(userUUID4),
		testUser(userUUID5),
	)

	createGroups(t, txn,
		testGroup(groupUUID1, []string{userUUID1}, nil, nil),
		testGroup(groupUUID2, []string{userUUID2}, nil, nil),
	)

	createRoles(t, txn,
		testRole("test", model.RoleScopeProject),
	)
}

func getTestMemoryStorage(t *testing.T, fixtureFunc func(t *testing.T, txn *io.MemoryStoreTxn)) *io.MemoryStore {
	// Create db with some entities.
	schema, err := repo.GetSchema()
	require.NoError(t, err)
	mem, err := io.NewMemoryStore(schema, nil)
	require.NoError(t, err)

	if fixtureFunc != nil {
		txn := mem.Txn(true)
		fixtureFunc(t, txn)
		err = txn.Commit()
		require.NoError(t, err)
	}

	return mem
}

func createUsers(t *testing.T, txn *io.MemoryStoreTxn, users ...*model.User) {
	for _, user := range users {
		tmp := user
		err := txn.Insert(model.UserType, tmp)
		require.NoError(t, err)
	}
}

func createGroups(t *testing.T, txn *io.MemoryStoreTxn, groups ...*model.Group) {
	for _, group := range groups {
		tmp := group
		err := txn.Insert(model.GroupType, tmp)
		require.NoError(t, err)
	}
}

func createRoleBindings(t *testing.T, txn *io.MemoryStoreTxn, items ...*model.RoleBinding) {
	for _, item := range items {
		tmp := item
		err := txn.Insert(model.RoleBindingType, tmp)
		require.NoError(t, err)
	}
}

func createRoles(t *testing.T, txn *io.MemoryStoreTxn, items ...*model.Role) {
	for _, item := range items {
		tmp := item
		err := txn.Insert(model.RoleType, tmp)
		require.NoError(t, err)
	}
}

func testUser(uuid string) *model.User {
	return &model.User{
		UUID:       uuid,
		TenantUUID: tenantUUID,
		Version:    "1",
		Identifier: uuid,
	}
}

func testGroup(uuid string, users []string, serviceAccounts []string, groups []string) *model.Group {
	return &model.Group{
		UUID:            uuid,
		TenantUUID:      tenantUUID,
		Version:         "1",
		Identifier:      uuid,
		Users:           users,
		ServiceAccounts: serviceAccounts,
		Groups:          groups,
	}
}

func testRole(name string, scope model.RoleScope) *model.Role {
	return &model.Role{Name: name, Scope: scope}
}
