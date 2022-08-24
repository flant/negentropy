package usecase

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_checkBackwardsCompatibility(t *testing.T) {
	oldSchemaJson := `{"type":"object","required":["name"],"properties":{"id":{"format":"int64","type":"integer"},"name":{"example":"doggie","pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"},"description":{"type":"string"}}}`
	err := checkRoleOptionSchema(oldSchemaJson)
	require.NoError(t, err)

	t.Run("new required option is prohibited", func(t *testing.T) {
		newSchemaJson := `{"type":"object","required":["name","id"],"properties":{"id":{"format":"int64","type":"integer"},"name":{"example":"doggie","pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"},"description":{"type":"string"}}}`
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err := checkBackwardsCompatibility(oldSchemaJson, newSchemaJson)

		require.Error(t, err)
	})

	t.Run("can't change type (property description)", func(t *testing.T) {
		newSchemaJson := `{"type":"object","required":["name"],"properties":{"id":{"format":"int64","type":"integer"},"name":{"example":"doggie","pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"},"description":{"type":"integer"}}}`
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err := checkBackwardsCompatibility(oldSchemaJson, newSchemaJson)

		require.Error(t, err)
	})

	t.Run("can't change format (property id)", func(t *testing.T) {
		newSchemaJson := `{"type":"object","required":["name"],"properties":{"id":{"format":"int32","type":"integer"},"name":{"example":"doggie","pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"},"description":{"type":"string"}}}`
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err := checkBackwardsCompatibility(oldSchemaJson, newSchemaJson)

		require.Error(t, err)
	})

	t.Run("can't change pattern (property name)", func(t *testing.T) {
		newSchemaJson := `{"type":"object","required":["name"],"properties":{"id":{"format":"int64","type":"integer"},"name":{"example":"doggie","pattern":"^[0-9a-f]{8}$","type":"string"},"description":{"type":"string"}}}`
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err := checkBackwardsCompatibility(oldSchemaJson, newSchemaJson)

		require.Error(t, err)
	})

	t.Run("can add new property", func(t *testing.T) {
		newSchemaJson := `{"type":"object","required":["name"],"properties":{"id":{"format":"int64","type":"integer"},"name":{"example":"doggie","pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"},"description":{"type":"string"}, "description2":{"type":"string"}}}`
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err := checkBackwardsCompatibility(oldSchemaJson, newSchemaJson)

		require.NoError(t, err)
	})

	t.Run("can change minor fields", func(t *testing.T) {
		newSchemaJson := `{"type":"object","required":["name"],"properties":{"id":{"format":"int64","type":"integer"},"name":{"example":"XXXXXX","pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"},"description":{"type":"string"}}}`
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err := checkBackwardsCompatibility(oldSchemaJson, newSchemaJson)

		require.NoError(t, err)
	})

	t.Run("can decrease required", func(t *testing.T) {
		newSchemaJson := `{"type":"object","required":[],"properties":{"id":{"format":"int64","type":"integer"},"name":{"example":"doggie","pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"},"description":{"type":"string"}}}`
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err := checkBackwardsCompatibility(oldSchemaJson, newSchemaJson)

		require.NoError(t, err)
	})

	t.Run("can add schema if not defined before without any required", func(t *testing.T) {
		newSchemaJson := `{"type":"object","required":[],"properties":{"id":{"format":"int64","type":"integer"},"name":{"example":"doggie","pattern":"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$","type":"string"},"description":{"type":"string"}}}`
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err := checkBackwardsCompatibility("", newSchemaJson)

		require.NoError(t, err)
	})

	t.Run("empty value for new is allowed, if old one was empty", func(t *testing.T) {
		newSchemaJson := ""
		require.NoError(t, checkRoleOptionSchema(newSchemaJson))

		err = checkBackwardsCompatibility("", newSchemaJson)

		require.NoError(t, err)
	})
}
