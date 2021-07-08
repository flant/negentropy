package flant_gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	storageKeyVaultRequestPrefix = "vault_request"
)

type vaultRequests []*vaultRequest

type vaultRequest struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Method  string `json:"method"`
	Options string `json:"options"`  // TODO: json type
	WrapTTL string `json:"wrap_ttl"` // TODO: golang duration type
}

func (b *backend) performWrappedVaultRequest(ctx context.Context, conf *vaultRequest) (string, error) {
	apiClient, err := b.AccessVaultController.APIClient()
	if err != nil {
		return "", fmt.Errorf("unable to get Vault API Client for Vault %q request to %q: %s", conf.Method, conf.Path, err)
	}

	request := apiClient.NewRequest(conf.Method, conf.Path)
	request.WrapTTL = conf.WrapTTL
	if conf.Options != "" {
		var data interface{}
		if err := json.Unmarshal([]byte(conf.Options), data); err != nil {
			panic(fmt.Sprintf("invalid configuration detected: unparsable options json string: %s\n%s\n", err, conf.Options))
		}

		if err := request.SetJSONBody(data); err != nil {
			return "", fmt.Errorf("error setting options json string into vault request: %s", err)
		}
	}

	resp, err := apiClient.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", fmt.Errorf("unable to perform vault request: %s", err)
	}

	defer resp.Body.Close()
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to parse secret from response: %s", err)
	}

	return secret.WrapInfo.Token, nil
}

func putVaultRequest(ctx context.Context, storage logical.Storage, vaultRequest vaultRequest) error {
	entry, err := logical.StorageEntryJSON(getAbsStoragePathToVaultRequest(vaultRequest.Name), vaultRequest)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, entry); err != nil {
		return err
	}

	return nil
}

func getVaultRequest(ctx context.Context, storage logical.Storage, vaultRequestName string) (*vaultRequest, error) {
	storageEntry, err := storage.Get(ctx, getAbsStoragePathToVaultRequest(vaultRequestName))
	if err != nil {
		return nil, err
	}
	if storageEntry == nil {
		return nil, nil
	}

	var request vaultRequest
	if err := storageEntry.DecodeJSON(request); err != nil {
		return nil, err
	}

	return &request, nil
}

func listVaultRequests(ctx context.Context, storage logical.Storage) (vaultRequests, error) {
	requestNames, err := storage.List(ctx, storageKeyVaultRequestPrefix)
	if err != nil {
		return nil, err
	}
	if len(requestNames) == 0 {
		return nil, nil
	}

	var requests vaultRequests
	for _, requestName := range requestNames {
		request, err := getVaultRequest(ctx, storage, requestName)
		if err != nil {
			return nil, err
		}
		if request == nil {
			continue
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func deleteVaultRequest(ctx context.Context, storage logical.Storage, vaultRequestName string) error {
	if err := storage.Delete(ctx, getAbsStoragePathToVaultRequest(vaultRequestName)); err != nil {
		return err
	}

	return nil
}

func getAbsStoragePathToVaultRequest(vaultRequestName string) string {
	return path.Join(storageKeyVaultRequestPrefix, vaultRequestName)
}
