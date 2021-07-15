package extension_server_access

import (
	"context"
	"sync"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"
)

const serverAccessConfigStorageKey = "iam.extensions.server_access_config"

var liveConfig = &mutexedConfig{}

type mutexedConfig struct {
	m sync.RWMutex

	configured bool
	sac        ServerAccessConfig
}

func (c *mutexedConfig) isConfigured() bool {
	return c.isConfigured()
}

func (c *mutexedConfig) GetServerAccessConfig(ctx context.Context, storage logical.Storage) (*ServerAccessConfig, error) {
	c.m.RLock()
	defer c.m.RUnlock()

	storedConfigEntry, err := storage.Get(ctx, serverAccessConfigStorageKey)
	if err != nil {
		return nil, err
	}
	if storedConfigEntry == nil {
		return nil, nil
	}

	var config ServerAccessConfig
	err = storedConfigEntry.DecodeJSON(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *mutexedConfig) SetServerAccessConfig(ctx context.Context, storage logical.Storage, config *ServerAccessConfig) error {
	encodedValue, err := jsonutil.EncodeJSON(*config)
	if err != nil {
		return err
	}

	err = storage.Put(ctx, &logical.StorageEntry{
		Key:   serverAccessConfigStorageKey,
		Value: encodedValue,
	})
	if err != nil {
		return err
	}

	c.configured = true

	return nil
}

func InitializeExtensionServerAccess(ctx context.Context, initRequest *logical.InitializationRequest) error {
	storage := initRequest.Storage

	config, err := liveConfig.GetServerAccessConfig(ctx, storage)
	if err != nil {
		return err
	}

	if config != nil {
		liveConfig.configured = true
	}

	return nil
}
