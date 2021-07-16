package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_LoadConfig(t *testing.T) {
	cfg, err := LoadConfig("testdata/conf.yaml")

	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "123-123-123", cfg.ServerAccessSettings.TenantUUID)
	require.Equal(t, "123-123-123", cfg.ServerAccessSettings.ProjectUUID)
	require.Equal(t, "123-123-123", cfg.ServerAccessSettings.ServerUUID)
	require.Equal(t, cfg.DatabasePath, "./server-accessd.db")
}
