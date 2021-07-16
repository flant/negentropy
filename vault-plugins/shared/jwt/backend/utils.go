package backend

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// protectNonEnabled protects the route handler if JWT issuing is not enabled
func protectNonEnabled(callback framework.OperationFunc) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		entry, err := req.Storage.Get(ctx, "jwt/enable")
		if err != nil {
			return nil, err
		}

		var enabled bool
		if entry != nil {
			err = entry.DecodeJSON(&enabled)
			if err != nil {
				return nil, err
			}
		}

		if !enabled {
			return nil, fmt.Errorf("jwt issuing is not enabled")
		}

		return callback(ctx, req, data)
	}
}
