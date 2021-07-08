package flant_gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/vault/api"
)

func (b *backend) performWrappedVaultRequest(ctx context.Context, conf *vaultRequest) (string, error) {
	apiClient, err := b.AccessVaultController.APIClient()
	if err != nil {
		return "", fmt.Errorf("unable to get Vault API Client for Vault %q request to %q: %s", conf.Method, conf.Path, err)
	}

	request := apiClient.NewRequest(conf.Method, conf.Path)
	request.WrapTTL = strconv.FormatFloat(conf.GetWrapTTL().Seconds(), 'f', 0, 64)
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
