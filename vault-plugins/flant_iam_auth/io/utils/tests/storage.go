package tests

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

func CreateTestStorage(t *testing.T) *io.MemoryStore {
	storageView := &logical.InmemStorage{}

	mb, err := sharedkafka.NewMessageBroker(context.TODO(), storageView, hclog.NewNullLogger())
	require.NoError(t, err)

	schema, err := repo.GetSchema()
	require.NoError(t, err)

	storage, err := io.NewMemoryStore(schema, mb, hclog.NewNullLogger())
	require.NoError(t, err)

	return storage
}
