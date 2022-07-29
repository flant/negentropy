package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
)

func Test_TenantList(t *testing.T) {
	tx := RunFixtures(t, TenantFixture).Txn(true)
	repo := iam_repo.NewTenantRepository(tx)

	tenants, err := repo.List(false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range tenants {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{fixtures.TenantUUID1, fixtures.TenantUUID2}, ids)
}
