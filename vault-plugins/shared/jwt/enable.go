package jwt

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func PathEnable(b *TokenController) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/enable`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleJWTEnableCreate,
				Summary:  pathJWTEnableSynopsis,
			},
		},

		HelpSynopsis:    pathJWTEnableSynopsis,
		HelpDescription: pathJWTEnableDesc,
	}
}

func PathDisable(b *TokenController) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/disable`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: protectNonEnabled(b.handleJWTDisableCreate),
				Summary:  pathJWTDisableSynopsis,
			},
		},

		HelpSynopsis:    pathJWTDisableSynopsis,
		HelpDescription: pathJWTDisableDesc,
	}
}

func switchJWT(ctx context.Context, req *logical.Request, data interface{}) (*logical.Response, error) {
	entry, err := logical.StorageEntryJSON("jwt/enable", data)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"enabled": data},
	}

	return resp, nil
}

func (b *TokenController) handleJWTEnableCreate(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	// put default config to the storage
	entry, err := req.Storage.Get(ctx, "jwt/configuration")
	if err != nil {
		return nil, err
	}

	if entry == nil {
		entry, err = logical.StorageEntryJSON("jwt/configuration", map[string]interface{}{
			"issuer":                      "https://auth.negentropy.flant.com/",
			"own_audience":                "",
			"preliminary_announce_period": "24h",
			"rotation_period":             "336h",
		})

		if err != nil {
			return nil, err
		}

		err = req.Storage.Put(ctx, entry)
		if err != nil {
			return nil, err
		}
	}

	// check that we have keys on first enabling
	entryPubs, err := req.Storage.Get(ctx, "jwt/jwks")
	if err != nil {
		return nil, err
	}

	entryPrivs, err := req.Storage.Get(ctx, "jwt/private_keys")
	if err != nil {
		return nil, err
	}

	if entryPubs == nil && entryPrivs == nil {
		err := generateOrRotateKeys(ctx, req.Storage)
		if err != nil {
			return nil, err
		}

		err = rotationTimestamp(ctx, req.Storage, b.now)
		if err != nil {
			return nil, err
		}
		return switchJWT(ctx, req, true)
	}

	if entryPubs != nil && entryPrivs != nil {
		return switchJWT(ctx, req, true)
	}

	return nil, fmt.Errorf("malformed jwt keys in the storage, try to re enable jwt authentication")
}

func (b *TokenController) handleJWTDisableCreate(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	err := req.Storage.Delete(ctx, "jwt/jwks")
	if err != nil {
		return nil, err
	}

	err = req.Storage.Delete(ctx, "jwt/private_keys")
	if err != nil {
		return nil, err
	}
	return switchJWT(ctx, req, false)
}

func (b *TokenController) IsEnabled(ctx context.Context, req *logical.Request) (bool, error) {
	isEnabledRaw, err := req.Storage.Get(ctx, "jwt/enable")
	if err != nil {
		return false, err
	}

	if isEnabledRaw == nil {
		return false, nil
	}

	var isEnabled bool
	err = isEnabledRaw.DecodeJSON(&isEnabled)
	if err != nil {
		return false, err
	}

	return isEnabled, nil
}

const (
	pathJWTEnableSynopsis = `
Enable JWT issuing.
`
	pathJWTEnableDesc = `
After enabling JWT issuing, users will be able to issue tokens with predefined claims and verify them using JWKS.
`

	pathJWTDisableSynopsis = `
Disable JWT issuing.
`
	pathJWTDisableDesc = `
Completely disable ability to issue JWT tokens for a vault.
`
)
