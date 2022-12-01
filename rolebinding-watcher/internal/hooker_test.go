package internal

import (
	"fmt"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/rolebinding-watcher/pkg"
	iam_fixtures "github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func RunFixtures(t *testing.T, store *io.MemoryStore, fixtures ...func(t *testing.T, store *io.MemoryStore)) *io.MemoryStore {
	for _, fixture := range fixtures {
		fixture(t, store)
	}
	return store
}

type mockProceeder struct {
	t             *testing.T
	expectedCalls []pkg.UserEffectiveRoles // put here items to check valid flow
	SkipCheck     bool
}

func (c *mockProceeder) ProceedUserEffectiveRole(newUsersEffectiveRoles pkg.UserEffectiveRoles) error {
	fmt.Printf("call %v\n", newUsersEffectiveRoles.Key()) // TODO REMOVE
	if c.SkipCheck {
		return nil
	}
	if len(c.expectedCalls) == 0 {
		c.t.Fatalf(fmt.Sprintf("mockProceeder is empty, but unexpected got: %#v", newUsersEffectiveRoles))
	}
	if newUsersEffectiveRoles.NotEqual(&c.expectedCalls[0]) {
		c.t.Fatalf(fmt.Sprintf("Expected: %#v\n got: %#v", c.expectedCalls[0], newUsersEffectiveRoles))
	}
	c.expectedCalls = c.expectedCalls[1:]
	return nil
}

func (c *mockProceeder) CallsToDo() []pkg.UserEffectiveRolesKey {
	if len(c.expectedCalls) == 0 {
		return nil
	}
	var result []pkg.UserEffectiveRolesKey
	for _, uer := range c.expectedCalls {
		result = append(result, uer.Key())
	}
	return result
}

func Test_Rolebindings(t *testing.T) {
	logger := hclog.NewNullLogger()
	store, err := memStorage(nil, logger)
	require.NoError(t, err)
	mock := &mockProceeder{t: t}
	tx := RunFixtures(t, store, iam_usecase.TenantFixture, iam_usecase.UserFixture, iam_usecase.ServiceAccountFixture, iam_usecase.GroupFixture, iam_usecase.ProjectFixture, iam_usecase.RoleFixture).Txn(true)
	require.NoError(t, tx.Commit())
	hooker := &Hooker{
		Logger: logger,
		processor: &ChangesProcessor{
			Logger:                     logger,
			userEffectiveRoleProcessor: mock,
		},
	}
	hooker.RegisterHooks(store)

	baseRolebinding := iam_model.RoleBinding{
		UUID:       iam_fixtures.RbUUID2,
		TenantUUID: iam_fixtures.TenantUUID2,
		Users:      []string{iam_fixtures.UserUUID1},
		Roles: []iam_model.BoundRole{{
			Name: iam_fixtures.RoleName1,
		}},
		AnyProject: true,
	}

	baseUserEffectiveRoles := pkg.UserEffectiveRoles{
		UserUUID: iam_fixtures.UserUUID1,
		RoleName: iam_fixtures.RoleName1,
		Tenants: []authz.EffectiveRoleTenantResult{{
			TenantUUID:       iam_fixtures.TenantUUID2,
			TenantIdentifier: "tenant2",
			TenantOptions:    map[string][]interface{}{},
			Projects: []authz.EffectiveRoleProjectResult{
				{
					ProjectUUID: iam_fixtures.ProjectUUID5, ProjectIdentifier: "pr5", ProjectOptions: map[string][]interface{}{}, RequireMFA: false, NeedApprovals: false,
				},
			},
		}},
	}

	t.Run("new rolebinding", func(t *testing.T) {
		mock.expectedCalls = []pkg.UserEffectiveRoles{baseUserEffectiveRoles}
		tx = store.Txn(true)
		rb := baseRolebinding

		require.NoError(t, tx.Insert(iam_model.RoleBindingType, &rb))
		require.NoError(t, tx.Commit())

		require.Nil(t, mock.CallsToDo())
	})

	t.Run("change rolebinding insignificant", func(t *testing.T) {
		mock.expectedCalls = []pkg.UserEffectiveRoles{} // empty calls
		tx = store.Txn(true)
		rb := baseRolebinding
		rb.Description = "Change insignificant filed"

		require.NoError(t, tx.Insert(iam_model.RoleBindingType, &rb))
		require.NoError(t, tx.Commit())

		require.Nil(t, mock.CallsToDo())
	})

	t.Run("change rolebinding significant", func(t *testing.T) {
		uer := baseUserEffectiveRoles
		uer.Tenants[0].TenantOptions = map[string][]interface{}{"k1": {"v1"}}
		mock.expectedCalls = []pkg.UserEffectiveRoles{uer}
		tx = store.Txn(true)
		rb := baseRolebinding
		rb.Roles[0].Options = map[string]interface{}{"k1": "v1"}

		require.NoError(t, tx.Insert(iam_model.RoleBindingType, &rb))
		require.NoError(t, tx.Commit())

		require.Nil(t, mock.CallsToDo())
	})

	t.Run("delete rolebinding", func(t *testing.T) {
		uer := baseUserEffectiveRoles
		uer.Tenants = nil // it means role disappears for a user
		mock.expectedCalls = []pkg.UserEffectiveRoles{uer}
		tx = store.Txn(true)
		rb := baseRolebinding
		rb.Archive(memdb.NewArchiveMark())

		require.NoError(t, tx.Insert(iam_model.RoleBindingType, &rb))
		require.NoError(t, tx.Commit())

		require.Nil(t, mock.CallsToDo())
	})
}

func Test_Groups(t *testing.T) {
	logger := hclog.NewNullLogger()
	store, err := memStorage(nil, logger)
	require.NoError(t, err)
	mock := &mockProceeder{t: t, SkipCheck: true}
	hooker := &Hooker{
		Logger: logger,
		processor: &ChangesProcessor{
			Logger:                     logger,
			userEffectiveRoleProcessor: mock,
		},
	}
	hooker.RegisterHooks(store)
	tx := RunFixtures(t, store, iam_usecase.TenantFixture, iam_usecase.UserFixture, iam_usecase.ServiceAccountFixture, iam_usecase.GroupFixture, iam_usecase.ProjectFixture, iam_usecase.RoleFixture).Txn(true)
	rolebinding := iam_model.RoleBinding{
		UUID:        iam_fixtures.RbUUID7,
		TenantUUID:  iam_fixtures.TenantUUID1,
		Description: "rb7",
		Groups:      []iam_model.GroupUUID{iam_fixtures.GroupUUID4},
		Roles: []iam_model.BoundRole{{
			Name: iam_fixtures.RoleName1,
		}},
		Origin: consts.OriginIAM,
	}
	tx.Insert(iam_model.RoleBindingType, &rolebinding)
	require.NoError(t, tx.Commit())

	userEffectiveRoles := pkg.UserEffectiveRoles{
		UserUUID: iam_fixtures.UserUUID5,
		RoleName: iam_fixtures.RoleName1,
		Tenants: []authz.EffectiveRoleTenantResult{{
			TenantUUID:       iam_fixtures.TenantUUID1,
			TenantIdentifier: "tenant1",
			TenantOptions:    map[string][]interface{}{},
		}},
	}

	t.Run("add user to group", func(t *testing.T) {
		mock.SkipCheck = false
		mock.expectedCalls = []pkg.UserEffectiveRoles{userEffectiveRoles}
		tx = store.Txn(true)
		group4, err := iam_repo.NewGroupRepository(tx).GetByID(iam_fixtures.GroupUUID4)
		var newGroup4 = *group4 // need create new object
		require.NoError(t, err)
		newGroup4.Users = append(group4.Users, iam_fixtures.UserUUID5)

		require.NoError(t, tx.Insert(iam_model.GroupType, &newGroup4))
		require.NoError(t, tx.Commit())

		require.Nil(t, mock.CallsToDo())
	})

	t.Run("delete user from group", func(t *testing.T) {
		uer := userEffectiveRoles
		uer.Tenants = nil // it means role disappears for a user
		mock.expectedCalls = []pkg.UserEffectiveRoles{uer}
		tx = store.Txn(true)
		group4, err := iam_repo.NewGroupRepository(tx).GetByID(iam_fixtures.GroupUUID4)
		var newGroup4 = *group4 // need create new object
		require.NoError(t, err)
		newGroup4.Users = group4.Users[0 : len(group4.Users)-1]

		require.NoError(t, tx.Insert(iam_model.GroupType, &newGroup4))
		require.NoError(t, tx.Commit())

		require.Nil(t, mock.CallsToDo())
	})
}
