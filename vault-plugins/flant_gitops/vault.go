package flant_gitops

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/logical"
)

func (b *backend) getVaultRequestWrapToken(ctx context.Context, conf *request) (string, *logical.Response, error) {
	apiClient, err := b.AccessVaultController.APIClient()
	if err != nil {
		return "", logical.ErrorResponse("Unable to get Vault API Client for Vault %q request to %q: %s", conf.Method, conf.Path, err), nil
	}

	request := apiClient.NewRequest(conf.Method, conf.Path)
	request.WrapTTL = strconv.FormatFloat(conf.GetWrapTTL().Seconds(), 'f', 0, 64)
	if conf.Options != "" {
		if err := request.SetJSONBody(conf.Options); err != nil {
			return "", nil, fmt.Errorf("Unable to convert options for Vault %q request to %q into JSON: %s", conf.Method, conf.Path, err)
		}
	}

	resp, err := apiClient.RawRequestWithContext(ctx, request)
	if err != nil {
		return "", logical.ErrorResponse("Unable to perform Vault %q request to %q: %s", conf.Method, conf.Path, err), nil
	}

	defer resp.Body.Close()
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("Unable to parse response to Vault %q request to %q: %s", conf.Method, conf.Path, err)
	}

	return secret.WrapInfo.Token, nil, nil
}
