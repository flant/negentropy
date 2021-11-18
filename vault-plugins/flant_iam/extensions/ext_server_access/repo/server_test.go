package repo

import (
	"testing"

	"github.com/stretchr/testify/require"

	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func Test_ServerDbSchema(t *testing.T) {
	iamSchema, err := iam_repo.GetSchema()
	require.NoError(t, err)

	_, err = memdb.MergeDBSchemas(iamSchema, ServerSchema())

	require.NoError(t, err)
}
