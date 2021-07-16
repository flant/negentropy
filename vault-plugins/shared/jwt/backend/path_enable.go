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
				Callback: protectNonEnabled(b.handleJWTDisableCreate),
				Summary:  pathJWTDisableSynopsis,
			},
		},

		HelpSynopsis:    pathJWTDisableSynopsis,
		HelpDescription: pathJWTDisableDesc,
	}
}

func (b *Backend) handleJWTEnableCreate(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	// put default config to the storage
	tnx := b.memStorage.Txn(true)
	defer tnx.Abort()

	s, err := b.deps.KeyPairsService(tnx)
	if err != nil {
		return nil, err
	}

	err = s.EnableJwt()
	if err != nil {
		return nil, err
	}

	if err := tnx.Commit(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *Backend) handleJWTDisableCreate(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	tnx := b.memStorage.Txn(true)
	defer tnx.Abort()

	s, err := b.deps.KeyPairsService(tnx)
	if err != nil {
		return nil, err
	}

	err = s.DisableJwt()
	if err != nil {
		return nil, err
	}

	if err := tnx.Commit(); err != nil {
		return nil, err
	}

	return nil, nil
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
