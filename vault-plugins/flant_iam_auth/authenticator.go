package jwtauth

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type Authenticator interface {
	Auth(ctx context.Context, d *framework.FieldData) (*logical.Auth, error)
}
