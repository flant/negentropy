package backend

import (
	"context"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func PathJWKS(b *Backend) *framework.Path {
	return &framework.Path{
		Pattern: `jwks`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.handleJWKSRead,
				Summary:  pathJWTJWKSSynopsis,
			},
		},

		HelpSynopsis: pathJWTJWKSSynopsis,
	}
}

func PathRotateKey(b *Backend) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/rotate_key`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleRotateKeysUpdate,
				Summary:  pathJWTRotateKeySynopsis,
			},
		},

		HelpSynopsis: pathJWTRotateKeySynopsis,
	}
}

func (b *Backend) handleJWKSRead(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	tnx := b.memStorage.Txn(false)
	defer tnx.Abort()

	err := b.mustEnabled(tnx)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	repo, err := b.deps.JwksRepo(tnx)
	if err != nil {
		return nil, err
	}

	keys, err := repo.GetSet()
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"keys": keys},
	}

	return resp, nil
}

func (b *Backend) handleRotateKeysUpdate(_ context.Context, _ *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	tnx := b.memStorage.Txn(true)
	defer tnx.Abort()

	err := b.mustEnabled(tnx)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	keyPairService, err := b.deps.KeyPairsService(tnx)
	if err != nil {
		return nil, err
	}

	err = keyPairService.ForceRotateKeys()
	if err != nil {
		return nil, err
	}

	if err := tnx.Commit(); err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"rotated": true},
	}

	return resp, nil
}

const (
	pathJWTJWKSSynopsis = `
Endpoint to expose public keys to check authority of issued tokens.
`
	pathJWTRotateKeySynopsis = `
Force key rotation. Calling this endpoint will rotate keys immediately.
`
)
