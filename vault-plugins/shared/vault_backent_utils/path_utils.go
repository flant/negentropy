package vault_backent_utils

import (
	"fmt"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"time"
)

func MissingParamErr(name string) *logical.Response {
	return logical.ErrorResponse("missing %v", name)
}

func NotEmptyStringParam(d *framework.FieldData, name string) (string, *logical.Response) {
	raw, ok := d.GetOk(name)
	val, okCast := raw.(string)
	if !ok || !okCast || val == "" {
		return "", MissingParamErr(name)
	}

	return val, nil
}

func DurationSecParam(d *framework.FieldData, name string, min int) (time.Duration, *logical.Response) {
	secretIdTtlRaw, ok := d.GetOk(name)
	var okCast bool
	secretIdTtlSec, okCast := secretIdTtlRaw.(int)
	if !ok || !okCast || secretIdTtlSec < min {
		return 0, logical.ErrorResponse(fmt.Sprintf("incorrect %s must be >= %vs", name, min))
	}

	return time.Duration(secretIdTtlSec), nil
}
