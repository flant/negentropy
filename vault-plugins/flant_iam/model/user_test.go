package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func Test_UserDbSchema(t *testing.T) {
	schema := UserSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("user schema is invalid: %v", err)
	}
}

func TestUserWithExtensions(t *testing.T) {
	u := User{
		UUID:           uuid.New(),
		TenantUUID:     uuid.New(),
		Version:        "1",
		Identifier:     "John",
		FullIdentifier: "test@John",
		Email:          "john@mail.com",
		Origin:         "test",
		Extensions: map[ObjectOrigin]*Extension{
			"test": {
				Origin:              "ext1",
				OwnerType:           "test",
				OwnerUUID:           uuid.New(),
				Attributes:          map[string]interface{}{"a": 1},
				SensitiveAttributes: map[string]interface{}{"b": 2},
			},
		},
	}

	t.Run("include sensitive data", func(t *testing.T) {
		data, err := json.Marshal(u)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"sensitive_attributes":{"b":2}`)
	})

	t.Run("exclude sensitive data", func(t *testing.T) {
		data, err := json.Marshal(OmitSensitive(u))
		require.NoError(t, err)
		assert.NotContains(t, string(data), `sensitive_attributes`)
	})
}
