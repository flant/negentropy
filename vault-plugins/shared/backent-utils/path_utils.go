package backentutils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
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

func ResponseErr(req *logical.Request, err error) (*logical.Response, error) {
	switch err {
	case consts.ErrNoUUID:
		return ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
	case consts.ErrNotFound:
		return ResponseErrMessage(req, err.Error(), http.StatusNotFound)
	case consts.ErrBadVersion:
		return ResponseErrMessage(req, err.Error(), http.StatusConflict)
	case consts.ErrBadOrigin, consts.ErrJwtDisabled:
		return ResponseErrMessage(req, err.Error(), http.StatusForbidden)
	case consts.ErrJwtControllerError:
		return ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	default:
		return nil, err
	}
}

func GetCreationID(expectID bool, data *framework.FieldData) (string, error) {
	if expectID {
		// for privileged access
		id := data.Get("uuid").(string)
		if id == "" {
			return "", consts.ErrNoUUID
		}
		return id, nil
	}

	return uuid.New(), nil
}
