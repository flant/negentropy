package client

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

func GetVaultClientConfig(ctx context.Context, storage logical.Storage) (*vaultAccessConfig, error) {
	if storage == nil {
		return nil, fmt.Errorf("%w: storage", consts.ErrNilPointer)
	}
	raw, err := storage.Get(ctx, storagePath)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	config := new(vaultAccessConfig)
	if err := raw.DecodeJSON(config); err != nil {
		return nil, err
	}

	return config, nil
}

func PutVaultClientConfig(ctx context.Context, conf *vaultAccessConfig, storage logical.Storage) error {
	if storage == nil {
		return fmt.Errorf("%w: storage", consts.ErrNilPointer)
	}
	entry, err := logical.StorageEntryJSON(storagePath, conf)
	if err != nil {
		return err
	}

	return storage.Put(ctx, entry)
}
