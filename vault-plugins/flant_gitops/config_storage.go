package flant_gitops

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

func getGitCredential(ctx context.Context, storage logical.Storage) (*gitCredential, error) {
	raw, err := storage.Get(ctx, storageKeyConfigurationGitCredential)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	config := new(gitCredential)
	if err := raw.DecodeJSON(config); err != nil {
		return nil, err
	}

	return config, nil
}

func putGitCredential(ctx context.Context, storage logical.Storage, raw map[string]interface{}) error {
	entry, err := logical.StorageEntryJSON(storageKeyConfigurationGitCredential, raw)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, entry); err != nil {
		return err
	}

	return err
}
