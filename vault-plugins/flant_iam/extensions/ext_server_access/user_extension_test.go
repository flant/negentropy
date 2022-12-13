package ext_server_access

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func dieOnErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		panic(err)
	}
}

func checkDeepEqual(t *testing.T, expected, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("\nExpected:\n%#v\nGot:\n%#v\n", expected, got)
	}
}

const (
	tenantUUID1         = "00000001-0000-4000-A000-000000000000"
	userUUID1           = "00000000-0000-4000-A000-000000000001"
	roleName1           = "ssh.open"
	roleName2           = "notSSH"
	rbUUID1             = "00000000-0000-4001-A000-000000000000"
	serviceAccountUUID1 = "00000000-0000-4000-A000-000000000011"
	groupUUID1          = "00000000-0001-4000-A000-000000000000"
)

func prepareTenant(t *testing.T, txn *io.MemoryStoreTxn) *model.Tenant {
	ten := &model.Tenant{
		UUID:       tenantUUID1,
		Identifier: "tenant1",
		Version:    "1",
	}
	err := txn.Insert(model.TenantType, ten)
	dieOnErr(t, err)
	return ten
}

func prepareSSHRole(t *testing.T, txn *io.MemoryStoreTxn) *model.Role {
	role := &model.Role{
		Name:        roleName1,
		Scope:       model.RoleScopeTenant,
		Description: roleName1,
	}
	err := txn.Insert(model.RoleType, role)
	dieOnErr(t, err)
	return role
}

func prepareNotSSHRole(t *testing.T, txn *io.MemoryStoreTxn) *model.Role {
	role := &model.Role{
		Name:        roleName2,
		Scope:       model.RoleScopeTenant,
		Description: roleName2,
	}
	err := txn.Insert(model.RoleType, role)
	dieOnErr(t, err)
	return role
}

func prepareUser(t *testing.T, txn *io.MemoryStoreTxn, tenantUUID model.TenantUUID) *model.User {
	user := &model.User{
		UUID:           userUUID1,
		TenantUUID:     tenantUUID,
		Version:        "1",
		Identifier:     "user1",
		FullIdentifier: "user1@test",
		Email:          "user@gmail.com",
	}
	err := txn.Insert(model.UserType, user)
	dieOnErr(t, err)
	return user
}

func prepareServiceAccount(t *testing.T, txn *io.MemoryStoreTxn, tenantUUID model.TenantUUID) *model.ServiceAccount {
	sa := &model.ServiceAccount{
		UUID:       serviceAccountUUID1,
		TenantUUID: tenantUUID,
		Version:    "1",
	}
	err := txn.Insert(model.ServiceAccountType, sa)
	dieOnErr(t, err)
	return sa
}

func prepareGroup(t *testing.T, txn *io.MemoryStoreTxn, tenantUUID model.TenantUUID,
	users []model.UserUUID, sas []model.ServiceAccountUUID, groups []model.GroupUUID) *model.Group {
	group := &model.Group{
		UUID:            groupUUID1,
		TenantUUID:      tenantUUID,
		Users:           users,
		Groups:          groups,
		ServiceAccounts: sas,
		FullIdentifier:  "group1@test",
		Origin:          "test",
		Identifier:      "group1",
	}
	err := txn.Insert(model.GroupType, group)
	dieOnErr(t, err)
	return group
}

func prepareRoleBinding(t *testing.T, txn *io.MemoryStoreTxn, roleName model.RoleName, tenantUUID model.TenantUUID,
	users []model.UserUUID, serviceAccounts []model.ServiceAccountUUID, groups []model.GroupUUID) *model.RoleBinding {
	rb := &model.RoleBinding{
		UUID:            rbUUID1,
		TenantUUID:      tenantUUID,
		Version:         "1",
		ValidTill:       0, // valid forever
		Users:           users,
		Groups:          groups,
		ServiceAccounts: serviceAccounts,
		AnyProject:      true,
		Roles: []model.BoundRole{{
			Name:    roleName,
			Options: nil,
		}},
		Description: "rolebinding1",
		Origin:      "test",
	}
	err := txn.Insert(model.RoleBindingType, rb)
	dieOnErr(t, err)
	return rb
}

func Test_UserIncludesToGroupWithSSH(t *testing.T) {
	schema, err := iam_repo.GetSchema()
	dieOnErr(t, err)
	store, err := io.NewMemoryStore(schema, nil, hclog.NewNullLogger())
	dieOnErr(t, err)
	txn := store.Txn(true)
	ten := prepareTenant(t, txn)
	role := prepareSSHRole(t, txn)
	user := prepareUser(t, txn, ten.UUID)
	// sa := prepareServiceAccount(t, txn, ten.UUID)
	group := prepareGroup(t, txn, ten.UUID, nil, nil, nil)
	prepareRoleBinding(t, txn, role.Name, ten.UUID, nil, nil, []model.GroupUUID{group.UUID})
	assert.NoError(t, err)
	storage := &logical.InmemStorage{}
	ctx := context.Background()
	err = liveConfig.SetServerAccessConfig(ctx, storage, &ServerAccessConfig{
		RolesForServers:                 nil,
		RoleForSSHAccess:                roleName1,
		DeleteExpiredPasswordSeedsAfter: 1000000,
		ExpirePasswordSeedAfterRevealIn: 1000000,
		LastAllocatedUID:                1,
	})
	repoUser := iam_repo.NewUserRepository(txn)
	user, err = repoUser.GetByID(user.UUID)
	dieOnErr(t, err)
	assert.NotContains(t, user.Extensions, consts.OriginServerAccess, "should not be an extension")

	RegisterServerAccessUserExtension(ctx, storage, store)
	newGroup := *group
	newGroup.Users = []model.UserUUID{user.UUID}
	err = txn.Insert(model.GroupType, &newGroup)

	dieOnErr(t, err)
	user, err = repoUser.GetByID(user.UUID)
	dieOnErr(t, err)
	assert.Contains(t, user.Extensions, consts.OriginServerAccess, "should be an extension")
	assert.Contains(t, user.Extensions[consts.OriginServerAccess].Attributes, "UID", "in extension should be UID")
	assert.Contains(t, user.Extensions[consts.OriginServerAccess].Attributes, "passwords", "in extension should be passwords")
	assert.Equal(t, 2, user.Extensions[consts.OriginServerAccess].Attributes["UID"], "actual UID should be defined as 1")
}

func Test_UserIsAddedToRoleBindingWithSSH(t *testing.T) {
	schema, err := iam_repo.GetSchema()
	dieOnErr(t, err)
	store, err := io.NewMemoryStore(schema, nil, hclog.NewNullLogger())
	dieOnErr(t, err)
	txn := store.Txn(true)
	ten := prepareTenant(t, txn)
	role := prepareSSHRole(t, txn)
	user := prepareUser(t, txn, ten.UUID)
	// sa := prepareServiceAccount(t, txn, ten.UUID)
	rb := prepareRoleBinding(t, txn, role.Name, ten.UUID, nil, nil, nil)
	assert.NoError(t, err)
	storage := &logical.InmemStorage{}
	ctx := context.Background()
	err = liveConfig.SetServerAccessConfig(ctx, storage, &ServerAccessConfig{
		RolesForServers:                 nil,
		RoleForSSHAccess:                roleName1,
		DeleteExpiredPasswordSeedsAfter: 1000000,
		ExpirePasswordSeedAfterRevealIn: 1000000,
		LastAllocatedUID:                1,
	})

	RegisterServerAccessUserExtension(ctx, storage, store)
	newRoleBinding := *rb
	newRoleBinding.Users = []model.UserUUID{user.UUID}
	err = txn.Insert(model.RoleBindingType, &newRoleBinding)

	dieOnErr(t, err)
	repoUser := iam_repo.NewUserRepository(txn)
	user, err = repoUser.GetByID(user.UUID)
	dieOnErr(t, err)
	assert.Contains(t, user.Extensions, consts.OriginServerAccess, "should be an extension")
	assert.Contains(t, user.Extensions[consts.OriginServerAccess].Attributes, "UID", "in extension should be UID")
	assert.Contains(t, user.Extensions[consts.OriginServerAccess].Attributes, "passwords", "in extension should be passwords")
	assert.Equal(t, 2, user.Extensions[consts.OriginServerAccess].Attributes["UID"], "actual UID should be defined as 1")
}

func Test_SSHIsAddedToRoleBinding(t *testing.T) {
	schema, err := iam_repo.GetSchema()
	dieOnErr(t, err)
	store, err := io.NewMemoryStore(schema, nil, hclog.NewNullLogger())
	dieOnErr(t, err)
	txn := store.Txn(true)
	ten := prepareTenant(t, txn)
	sshRole := prepareSSHRole(t, txn)
	notSSHRRole := prepareNotSSHRole(t, txn)
	user := prepareUser(t, txn, ten.UUID)
	// sa := prepareServiceAccount(t, txn, ten.UUID)
	rb := prepareRoleBinding(t, txn, notSSHRRole.Name, ten.UUID, []model.UserUUID{user.UUID}, nil, nil)
	assert.NoError(t, err)
	storage := &logical.InmemStorage{}
	ctx := context.Background()
	err = liveConfig.SetServerAccessConfig(ctx, storage, &ServerAccessConfig{
		RolesForServers:                 nil,
		RoleForSSHAccess:                roleName1,
		DeleteExpiredPasswordSeedsAfter: 1000000,
		ExpirePasswordSeedAfterRevealIn: 1000000,
		LastAllocatedUID:                1,
	})

	RegisterServerAccessUserExtension(ctx, storage, store)
	newRoleBinding := *rb
	newRoleBinding.Roles = []model.BoundRole{{
		Name:    sshRole.Name,
		Options: nil,
	}}
	err = txn.Insert(model.RoleBindingType, &newRoleBinding)

	dieOnErr(t, err)
	repoUser := iam_repo.NewUserRepository(txn)
	user, err = repoUser.GetByID(user.UUID)
	dieOnErr(t, err)
	assert.Contains(t, user.Extensions, consts.OriginServerAccess, "should be an extension")
	assert.Contains(t, user.Extensions[consts.OriginServerAccess].Attributes, "UID", "in extension should be UID")
	assert.Contains(t, user.Extensions[consts.OriginServerAccess].Attributes, "passwords", "in extension should be passwords")
	assert.Equal(t, 2, user.Extensions[consts.OriginServerAccess].Attributes["UID"], "actual UID should be defined as 1")
}

func Test_SSHIsIncludedToRole(t *testing.T) {
	schema, err := iam_repo.GetSchema()
	dieOnErr(t, err)
	store, err := io.NewMemoryStore(schema, nil, hclog.NewNullLogger())
	dieOnErr(t, err)
	txn := store.Txn(true)
	ten := prepareTenant(t, txn)
	sshRole := prepareSSHRole(t, txn)
	notSSHRRole := prepareNotSSHRole(t, txn)
	user := prepareUser(t, txn, ten.UUID)
	// sa := prepareServiceAccount(t, txn, ten.UUID)
	prepareRoleBinding(t, txn, notSSHRRole.Name, ten.UUID, []model.UserUUID{user.UUID}, nil, nil)
	assert.NoError(t, err)
	storage := &logical.InmemStorage{}
	ctx := context.Background()
	err = liveConfig.SetServerAccessConfig(ctx, storage, &ServerAccessConfig{
		RolesForServers:                 nil,
		RoleForSSHAccess:                roleName1,
		DeleteExpiredPasswordSeedsAfter: 1000000,
		ExpirePasswordSeedAfterRevealIn: 1000000,
		LastAllocatedUID:                1,
	})

	RegisterServerAccessUserExtension(ctx, storage, store)
	newNotSSHRole := *notSSHRRole
	newNotSSHRole.IncludedRoles = []model.IncludedRole{{
		Name: sshRole.Name,
	}}
	err = txn.Insert(model.RoleType, &newNotSSHRole)

	dieOnErr(t, err)
	repoUser := iam_repo.NewUserRepository(txn)
	user, err = repoUser.GetByID(user.UUID)
	dieOnErr(t, err)
	assert.Contains(t, user.Extensions, consts.OriginServerAccess, "should be an extension")
	assert.Contains(t, user.Extensions[consts.OriginServerAccess].Attributes, "UID", "in extension should be UID")
	assert.Contains(t, user.Extensions[consts.OriginServerAccess].Attributes, "passwords", "in extension should be passwords")
	assert.Equal(t, 2, user.Extensions[consts.OriginServerAccess].Attributes["UID"], "actual UID should be defined as 1")
}
