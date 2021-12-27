package authn

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// Result represents results of applying algorithm relying on auth_method_type and auth_method params on
// passed login params
type Result struct {
	// identifier of service account or user
	UUID string
	// empty is unknown
	ModelType string

	// for audit log
	Metadata map[string]string

	// for renew
	InternalData map[string]interface{}

	// TODO ???
	Policies []string

	// TODO ???
	GroupAliases []string

	// TODO ???
	Claims map[string]interface{}
}

type Authenticator interface {
	Authenticate(ctx context.Context, d *framework.FieldData) (*Result, error)
	CanRenew(vaultAuth *logical.Auth) (bool, error)
}
