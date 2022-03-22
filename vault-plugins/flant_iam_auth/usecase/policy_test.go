package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func createTenants(t *testing.T, repo *repo.PolicyRepository, policies ...model.Policy) {
	for _, policy := range policies {
		tmp := policy
		err := repo.Create(&tmp)
		require.NoError(t, err)
	}
}

func policyFixture(t *testing.T, store *io.MemoryStore) {
	tx := store.Txn(true)
	repo := repo.NewPolicyRepository(tx)
	createTenants(t, repo, fixtures.Policies()...)
	err := tx.Commit()
	require.NoError(t, err)
}

func Test_PolicyList(t *testing.T) {
	tx := runFixtures(t, roleFixture, policyFixture).Txn(true)
	service := Policies(tx)

	policies, err := service.List(false)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{fixtures.PolicyName1, fixtures.PolicyName2}, policies)
}
