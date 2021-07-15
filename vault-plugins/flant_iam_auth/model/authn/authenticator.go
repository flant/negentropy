package authn

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
)

type Result struct {
	// identifier of service account or user
	UUID string
	// empty is unknown
	ModelType string

	Metadata     map[string]string
	Policies     []string
	GroupAliases []string
}

type Authenticator interface {
	Authenticate(ctx context.Context, d *framework.FieldData) (*Result, error)
}
