package backentutils

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func BigBadabumPath(b *framework.Backend) []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "bigbadabum",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback:    handleListAvailableRoles(b),
					Summary:     "Produce big badabum at the plugin",
					Description: "Cause panic at the plugin",
				},
			},
		},
	}
}

func handleListAvailableRoles(b *framework.Backend) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		(*b).Logger().Named("Big_Badabum")
		message := "Big badabum is called, plugin will panic"
		(*b).Logger().Error(message)
		panic(message)
	}
}
