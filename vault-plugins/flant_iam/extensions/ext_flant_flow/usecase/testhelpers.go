package usecase

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

func runFixtures(t *testing.T, fixtures ...func(t *testing.T, store *io.MemoryStore)) *io.MemoryStore {
	schema, err := repo.GetSchema()
	require.NoError(t, err)
	iamSchema, err := iam_repo.GetSchema()
	require.NoError(t, err)
	schema, err = memdb.MergeDBSchemas(schema, iamSchema)

	store, err := io.NewMemoryStore(schema, nil, hclog.NewNullLogger())
	require.NoError(t, err)
	for _, fixture := range fixtures {
		fixture(t, store)
	}
	return store
}
