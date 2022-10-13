package client

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

func (c *VaultClientController) getVaultClientConfig(ctx context.Context) (*vaultAccessConfig, error) {
	raw, err := c.storage.Get(ctx, storagePath)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotSetConf
	}

	config := new(vaultAccessConfig)
	if err := raw.DecodeJSON(config); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *VaultClientController) saveVaultClientConfig(ctx context.Context, conf *vaultAccessConfig) error {
	entry, err := logical.StorageEntryJSON(storagePath, conf)
	if err != nil {
		return err
	}

	return c.storage.Put(ctx, entry)
}
