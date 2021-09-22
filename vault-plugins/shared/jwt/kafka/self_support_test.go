package kafka

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
)

func Test_SelfRestoreMessage_JWTConfigType(t *testing.T) {
	schema := model.ConfigSchema()
	store, err := io.NewMemoryStore(schema, nil)
	require.NoError(t, err)
	txn := store.Txn(true)

	handled, err := SelfRestoreMessage(txn.Txn, model.JWTConfigType,
		[]byte(`{
   "id": "jwt_config",
   "config": {
      "issuer": "https://auth.negentropy.flant.com/",
      "multipass_audience": "",
      "rotation_period": 1209600000000000,
      "preliminary_announce_period": 86400000000000
   }
}
`))

	require.NoError(t, err)
	require.True(t, handled)
	repo := model.NewConfigRepo(txn)
	cfg, err := repo.Get()
	require.NoError(t, err)
	require.Equal(t, "https://auth.negentropy.flant.com/", cfg.Issuer)
	require.Equal(t, time.Duration(1209600000000000), cfg.RotationPeriod)
	require.Equal(t, time.Duration(86400000000000), cfg.PreliminaryAnnouncePeriod)
}
