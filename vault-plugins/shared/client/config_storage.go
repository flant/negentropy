package client

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

type accessConfigStorage struct {
	parent logical.Storage
}

func newAccessConfigStorage(parent logical.Storage) *accessConfigStorage {
	return &accessConfigStorage{parent: parent}
}

func (s *accessConfigStorage) Get(ctx context.Context) (*vaultAccessConfig, error) {
	raw, err := s.parent.Get(ctx, storagePath)
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

func (s *accessConfigStorage) Put(ctx context.Context, conf *vaultAccessConfig) error {
	entry, err := logical.StorageEntryJSON(storagePath, conf)
	if err != nil {
		return err
	}

	return s.parent.Put(ctx, entry)
}
