package internal

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func RunFixtures(t *testing.T, store *io.MemoryStore, fixtures ...func(t *testing.T, store *io.MemoryStore)) *io.MemoryStore {
	for _, fixture := range fixtures {
		fixture(t, store)
	}
	return store
}

func Test_nnn(t *testing.T) {
	logger := hclog.Default()
	logger.SetLevel(hclog.Trace)
	store, err := memStorage(nil, logger)
	require.NoError(t, err)

	hooker := &Hooker{
		Logger: logger,
		processor: &ChangesProcessor{
			userEffectiveRoleProcessor: MockProceeder{},
		}}
	hooker.RegisterHooks(store)

	tx := RunFixtures(t, store, iam_usecase.TenantFixture, iam_usecase.UserFixture, iam_usecase.ServiceAccountFixture, iam_usecase.GroupFixture, iam_usecase.ProjectFixture, iam_usecase.RoleFixture,
		iam_usecase.RoleBindingFixture).Txn(true)
	err = tx.Commit()
	if err != nil {
		panic(err)
	}

	RunFixtures(t, store, iam_usecase.RoleBindingFixture).Txn(true)
	err = tx.Commit()
	if err != nil {
		panic(err)
	}

}
