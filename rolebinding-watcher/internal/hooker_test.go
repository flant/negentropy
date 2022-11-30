package internal

import (
	"fmt"
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
	t                           *testing.T
	expectedUsersEffectiveRoles []pkg.UserEffectiveRoles // put here items to check valid flow
}

func (c *mockProceeder) ProceedUserEffectiveRole(newUsersEffectiveRoles pkg.UserEffectiveRoles) error {
	if c.Empty() {
		c.t.Fatalf(fmt.Sprintf("mockProceeder is empty, but Unexpected got: %#v", newUsersEffectiveRoles))
	}
	if newUsersEffectiveRoles.NotEqual(&c.expectedUsersEffectiveRoles[0]) {
		c.t.Fatalf(fmt.Sprintf("Expected: %#v\n got: %#v", c.expectedUsersEffectiveRoles[0], newUsersEffectiveRoles))
	}
	c.expectedUsersEffectiveRoles = c.expectedUsersEffectiveRoles[1:]
	return nil
}

func (c *mockProceeder) Empty() bool {
	return len(c.expectedUsersEffectiveRoles) == 0
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
		}}
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
		Tenants: []authz.EffectiveRoleTenantResult{authz.EffectiveRoleTenantResult{
			TenantUUID:       iam_fixtures.TenantUUID2,
			TenantIdentifier: "tenant2",
			TenantOptions:    map[string][]interface{}{},
			Projects: []authz.EffectiveRoleProjectResult{authz.EffectiveRoleProjectResult{
				ProjectUUID: iam_fixtures.ProjectUUID5, ProjectIdentifier: "pr5", ProjectOptions: map[string][]interface{}{}, RequireMFA: false, NeedApprovals: false},
			},
		}},
	}

	t.Run("new rolebinding", func(t *testing.T) {
		mock.expectedUsersEffectiveRoles = []pkg.UserEffectiveRoles{baseUserEffectiveRoles}
		tx = store.Txn(true)
		rb := baseRolebinding

		require.NoError(t, tx.Insert(iam_model.RoleBindingType, &rb))
		require.NoError(t, tx.Commit())

		require.Equal(t, true, mock.Empty())
	})

	t.Run("change rolebinding insignificant", func(t *testing.T) {
		mock.expectedUsersEffectiveRoles = []pkg.UserEffectiveRoles{} // empty calls
		tx = store.Txn(true)
		rb := baseRolebinding
		rb.Description = "Change insignificant filed"

		require.NoError(t, tx.Insert(iam_model.RoleBindingType, &rb))
		require.NoError(t, tx.Commit())

		require.Equal(t, true, mock.Empty())
	})

	t.Run("change rolebinding significant", func(t *testing.T) {
		uer := baseUserEffectiveRoles
		uer.Tenants[0].TenantOptions = map[string][]interface{}{"k1": {"v1"}}
		mock.expectedUsersEffectiveRoles = []pkg.UserEffectiveRoles{uer}
		tx = store.Txn(true)
		rb := baseRolebinding
		rb.Roles[0].Options = map[string]interface{}{"k1": "v1"}

		require.NoError(t, tx.Insert(iam_model.RoleBindingType, &rb))
		require.NoError(t, tx.Commit())

		require.Equal(t, true, mock.Empty())
	})

	t.Run("delete rolebinding", func(t *testing.T) {
		uer := baseUserEffectiveRoles
		uer.Tenants = nil // it means role disappears for a user
		mock.expectedUsersEffectiveRoles = []pkg.UserEffectiveRoles{uer}
		tx = store.Txn(true)
		rb := baseRolebinding
		rb.Archive(memdb.NewArchiveMark())

		require.NoError(t, tx.Insert(iam_model.RoleBindingType, &rb))
		require.NoError(t, tx.Commit())

		require.Equal(t, true, mock.Empty())
	})
}
