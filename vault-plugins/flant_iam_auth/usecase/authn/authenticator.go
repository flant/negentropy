package authn

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type Result struct {
	// identifier of service account or user
	UUID string
	// empty is unknown
	ModelType string

	// for audit log
	Metadata map[string]string
	// for renew
	InternalData map[string]interface{}

	Policies     []string
	GroupAliases []string

	Claims map[string]interface{}
}

type Authenticator interface {
	Authenticate(ctx context.Context, d *framework.FieldData) (*Result, error)
	CanRenew(vaultAuth *logical.Auth) (bool, error)
}
