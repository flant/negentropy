package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_LoadConfig(t *testing.T) {
	cfg, err := LoadConfig("testdata/conf.yaml")

	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, cfg.ServerAccessSettings.TenantUUID, "123-123-123")
	require.Equal(t, cfg.DatabasePath, "./server-accessd.db")
}
