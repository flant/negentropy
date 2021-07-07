package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func TestMarshalling(t *testing.T) {
	ex := &Extension{
		Origin:              "test",
		OwnerType:           "test",
		OwnerUUID:           uuid.New(),
		Attributes:          map[string]interface{}{"a": 1},
		SensitiveAttributes: map[string]interface{}{"b": 2},
	}

	t.Run("with sensitive", func(t *testing.T) {
		data, err := ex.Marshal(true)
		require.NoError(t, err)
		assert.Contains(t, string(data), "sensitive_attributes")
	})

	t.Run("exclude sensitive", func(t *testing.T) {
		data, err := ex.Marshal(false)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "sensitive_attributes")
	})
}
