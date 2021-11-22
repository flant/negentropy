package usecase

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func runFixtures(t *testing.T, fixtures ...func(t *testing.T, store *io.MemoryStore)) *io.MemoryStore {
	schema, err := repo.GetSchema()
	require.NoError(t, err)
	store, err := io.NewMemoryStore(schema, nil, hclog.NewNullLogger())
	require.NoError(t, err)
	for _, fixture := range fixtures {
		fixture(t, store)
	}
	return store
}
