package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
)

func Test_ServiceAccountList(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, ServiceAccountFixture).Txn(true)
	repo := iam_repo.NewServiceAccountRepository(tx)

	serviceAccounts, err := repo.List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range serviceAccounts {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.ServiceAccountUUID1, fixtures.ServiceAccountUUID2, fixtures.ServiceAccountUUID3}, ids)
}
