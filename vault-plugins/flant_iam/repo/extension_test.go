package repo

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func Test_ExtensionMarshalling(t *testing.T) {
	ex := &model.Extension{
		Origin:              "test",
		OwnerType:           "test",
		OwnerUUID:           uuid.New(),
		Attributes:          map[string]interface{}{"a": 1},
		SensitiveAttributes: map[string]interface{}{"b": 2},
	}

	t.Run("with sensitive", func(t *testing.T) {
		data, err := json.Marshal(ex)
		require.NoError(t, err)
		assert.Contains(t, string(data), "sensitive_attributes")
	})

	t.Run("exclude sensitive", func(t *testing.T) {
		data, err := json.Marshal(OmitSensitive(ex))
		require.NoError(t, err)
		assert.NotContains(t, string(data), "sensitive_attributes")
	})
}
