package extension_server_access

import (
	"context"
	"sync"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
)

const serverAccessConfigStorageKey = "iam.extensions.server_access_config"

type mutexedConfig struct {
	m sync.RWMutex

	isConfigured bool
	sac          ServerAccessConfig
}

var liveConfig = &mutexedConfig{}

func InitializeExtensionServerAccess(ctx context.Context, initRequest *logical.InitializationRequest) error {
	storage := initRequest.Storage

	sac, err := GetServerAccessConfig(ctx, storage)
	if err != nil {
		return err
	}

	SetServerAccessConfig()
}

func GetServerAccessConfig(ctx context.Context, storage logical.Storage) (ServerAccessConfig, error) {
	storedConfigEntry, err := storage.Get(ctx, serverAccessConfigStorageKey)
	if err != nil {
		return ServerAccessConfig{}, err
	}

	var config ServerAccessConfig
	err = storedConfigEntry.DecodeJSON(&config)
	if err != nil {
		return ServerAccessConfig{}, err
	}

	return config, nil
}

func SetServerAccessConfig(ctx context.Context, storage logical.Storage, config ServerAccessConfig) error {
	encodedValue, err := jsonutil.EncodeJSON(config)
	if err != nil {
		return err
	}

	return storage.Put(ctx, &logical.StorageEntry{
		Key:   serverAccessConfigStorageKey,
		Value: encodedValue,
	})
}
