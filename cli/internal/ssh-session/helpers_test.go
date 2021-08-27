package ssh_session

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flant/negentropy/cli/internal/model"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

const path = "test.json"

var sl = model.ServerList{
	Tenants: map[iam.TenantUUID]iam.Tenant{"t1": {
		UUID:       "t1",
		Version:    "v1",
		Identifier: "i1",
	}},
	Projects: map[iam.ProjectUUID]iam.Project{"p1": {
		UUID:       "p1",
		TenantUUID: "t1",
		Version:    "v1",
		Identifier: "i2",
	}},
	Servers: map[ext.ServerUUID]ext.Server{"s1": {
		UUID:          "s1",
		TenantUUID:    "t1",
		ProjectUUID:   "p1",
		Version:       "v1",
		Identifier:    "i3",
		MultipassUUID: "m1",
		Fingerprint:   "f1",
	}},
}

func TestSaveToFile(t *testing.T) {
	err := deleteFileIfExists(path)
	require.NoError(t, err)

	err = SaveToFile(sl, path)

	require.NoError(t, err)
	require.FileExists(t, path)
	deleteFileIfExists(path)
}

func TestReadFromFile(t *testing.T) {
	err := deleteFileIfExists(path)
	require.NoError(t, err)
	err = SaveToFile(sl, path)
	require.NoError(t, err)
	require.FileExists(t, path)

	serverList, err := ReadFromFile(path)

	require.NoError(t, err)
	require.EqualValues(t, sl, *serverList)
	deleteFileIfExists(path)
}

func deleteFileIfExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}
