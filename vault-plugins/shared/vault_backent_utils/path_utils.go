package vault_backent_utils

import (
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func MissingParamErr(name string) *logical.Response {
	return logical.ErrorResponse("missing %v", name)
}

func NotEmptyStringParam(d *framework.FieldData, name string) (string, *logical.Response) {
	raw, ok := d.GetOk(name)
	val, okCast := raw.(string)
	if !ok || !okCast || val == "" {
		return "", MissingParamErr("role_name")
	}

	return val, nil
}
