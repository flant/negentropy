package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

func Test_ProjectList(t *testing.T) {
	tx := RunFixtures(t, TenantFixture, ProjectFixture).Txn(true)

	projects, err := Projects(tx, consts.OriginIAM).List(fixtures.TenantUUID1, false)

	require.NoError(t, err)
	ids := make([]string, 0)
	for _, obj := range projects {
		ids = append(ids, obj.ObjId())
	}
	require.ElementsMatch(t, []string{
		fixtures.ProjectUUID1, fixtures.ProjectUUID2, fixtures.ProjectUUID3, fixtures.ProjectUUID4,
	}, ids)
}
