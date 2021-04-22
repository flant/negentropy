package jwtauth

import (
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func nameFromRequest(d *framework.FieldData) (string, *logical.Response) {
	sourceName := d.Get("name").(string)
	if sourceName == "" {
		return "", logical.ErrorResponse("name is required")
	}

	return sourceName, nil
}
