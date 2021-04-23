package jwt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// newUUID generates random string
func newUUID() string {
	u := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, u); err != nil {
		panic(err)
	}

	u[8] = (u[8] | 0x80) & 0xBF
	u[6] = (u[6] | 0x40) & 0x4F

	return hex.EncodeToString(u)
}

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
