package usecase

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func RunFixtures(t *testing.T, fixtures ...func(t *testing.T, store *io.MemoryStore)) *io.MemoryStore {
	schema, err := iam_repo.GetSchema()
	require.NoError(t, err)
	store, err := io.NewMemoryStore(schema, nil, hclog.NewNullLogger())
	require.NoError(t, err)
	for _, fixture := range fixtures {
		fixture(t, store)
	}
	return store
}

func createTenants(t *testing.T, repo *iam_repo.TenantRepository, tenants ...model.Tenant) {
	for _, tenant := range tenants {
		tmp := tenant
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func TenantFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewTenantRepository(tx)
	createTenants(t, repo, fixtures.Tenants()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func createProjects(t *testing.T, repo *ProjectService, projects ...model.Project) {
	for _, project := range projects {
		tmp := project
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func ProjectFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := Projects(tx, consts.OriginIAM)
	createProjects(t, repo, fixtures.Projects()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func createRoles(t *testing.T, repo *iam_repo.RoleRepository, roles ...model.Role) {
	for _, role := range roles {
		tmp := role
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func RoleFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewRoleRepository(tx)
	createRoles(t, repo, fixtures.Roles()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func createUsers(t *testing.T, repo *iam_repo.UserRepository, users ...model.User) {
	for _, user := range users {
		tmp := user
		tmp.Version = uuid.New()
		tmp.FullIdentifier = "user_" + uuid.New()
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func UserFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewUserRepository(tx)
	createUsers(t, repo, fixtures.Users()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func createServiceAccounts(t *testing.T, repo *iam_repo.ServiceAccountRepository, sas ...model.ServiceAccount) {
	for _, sa := range sas {
		tmp := sa
		tmp.FullIdentifier = "service_account_" + uuid.New() // delete after bringing full identifiers to usecases
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func ServiceAccountFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := iam_repo.NewServiceAccountRepository(tx)
	createServiceAccounts(t, repo, fixtures.ServiceAccounts()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func createGroups(t *testing.T, repo *iam_repo.GroupRepository, groups ...model.Group) {
	for _, group := range groups {
		tmp := group
		tmp.FullIdentifier = "group_" + uuid.New()
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func GroupFixture(t *testing.T, store *io.MemoryStore) {
	gs := fixtures.Groups()
	for i := range gs {
		gs[i].Members = appendMembers(makeMemberNotations(model.UserType, gs[i].Users),
			makeMemberNotations(model.ServiceAccountType, gs[i].ServiceAccounts),
			makeMemberNotations(model.GroupType, gs[i].Groups))
	}
	tx := store.Txn(true)
	repository := iam_repo.NewGroupRepository(tx)
	createGroups(t, repository, gs...)
	err := tx.Commit()
	require.NoError(t, err)
}

func createRoleBindings(t *testing.T, repo *iam_repo.RoleBindingRepository, rbs ...model.RoleBinding) {
	for _, rb := range rbs {
		tmp := rb
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func RoleBindingFixture(t *testing.T, store *io.MemoryStore) {
	rbs := fixtures.RoleBindings()
	for i := range rbs {
		rbs[i].Members = appendMembers(makeMemberNotations(model.UserType, rbs[i].Users),
			makeMemberNotations(model.ServiceAccountType, rbs[i].ServiceAccounts),
			makeMemberNotations(model.GroupType, rbs[i].Groups))
	}
	tx := store.Txn(true)
	repo := iam_repo.NewRoleBindingRepository(tx)
	createRoleBindings(t, repo, rbs...)
	err := tx.Commit()
	require.NoError(t, err)
}

func toMemberNotation(m iam_repo.Model) model.MemberNotation {
	return model.MemberNotation{
		Type: m.ObjType(),
		UUID: m.ObjId(),
	}
}

func toMemberNotations(ms ...iam_repo.Model) []model.MemberNotation {
	sns := make([]model.MemberNotation, 0)
	for _, m := range ms {
		sns = append(sns, toMemberNotation(m))
	}
	return sns
}

func makeMemberNotations(memberType string, uuids []string) []model.MemberNotation {
	validTypes := map[string]struct{}{model.ServiceAccountType: {}, model.UserType: {}, model.GroupType: {}}
	if _, valid := validTypes[memberType]; !valid {
		panic(fmt.Errorf("member_type %s is invalid", memberType)) // nolint:panic_check
	}
	result := make([]model.MemberNotation, len(uuids))
	for i := range uuids {
		result[i] = model.MemberNotation{
			Type: memberType,
			UUID: uuids[i],
		}
	}
	return result
}

func appendMembers(membersGroups ...[]model.MemberNotation) []model.MemberNotation {
	result := []model.MemberNotation{}
	for i := range membersGroups {
		result = append(result, membersGroups[i]...)
	}
	return result
}
