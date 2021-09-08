package backend

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func PathEnable(b *Backend) *framework.Path {
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

func PathDisable(b *Backend) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/disable`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleJWTDisableCreate,
				Summary:  pathJWTDisableSynopsis,
			},
		},

		HelpSynopsis:    pathJWTDisableSynopsis,
		HelpDescription: pathJWTDisableDesc,
	}
}

func (b *Backend) handleJWTEnableCreate(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	// put default config to the storage
	txn := b.memStorage.Txn(true)
	defer txn.Abort()

	s, err := b.deps.KeyPairsService(txn)
	if err != nil {
		return nil, err
	}

	err = s.EnableJwt()
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"enabled": true},
	}

	return resp, nil
}

func (b *Backend) handleJWTDisableCreate(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	txn := b.memStorage.Txn(true)
	defer txn.Abort()

	s, err := b.deps.KeyPairsService(txn)
	if err != nil {
		return nil, err
	}

	err = s.DisableJwt()
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"enabled": false},
	}

	return resp, nil
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
