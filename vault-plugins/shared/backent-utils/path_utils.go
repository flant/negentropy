package backentutils

import (
	"fmt"
	"time"

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
		return "", MissingParamErr(name)
	}

	return val, nil
}

func DurationSecParam(d *framework.FieldData, name string, min int) (time.Duration, *logical.Response) {
	raw, ok := d.GetOk(name)
	var okCast bool
	val, okCast := raw.(int)
	if !ok || !okCast || val < min {
		return 0, logical.ErrorResponse(fmt.Sprintf("incorrect %s must be >= %vs", name, min))
	}

	return time.Duration(val), nil
}

func ResponseErrMessage(req *logical.Request, message string, status int) (*logical.Response, error) {
	rr := logical.ErrorResponse(message)
	return logical.RespondWithStatusCode(rr, req, status)
}
