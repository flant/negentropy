package backentutils

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/pkg/errors"

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
	statusCode := MapErrorToHTTPStatusCode(err)
	if statusCode == 0 {
		return nil, err
	}
	return ResponseErrMessage(req, err.Error(), statusCode)
}

func MapErrorToHTTPStatusCode(err error) int {
	statuses := map[error]int{
		consts.ErrNoUUID:     http.StatusBadRequest,
		consts.ErrIsArchived: http.StatusBadRequest,

		consts.ErrNotFound: http.StatusNotFound,

		consts.ErrBadVersion: http.StatusConflict,

		consts.ErrBadOrigin:   http.StatusForbidden,
		consts.ErrJwtDisabled: http.StatusForbidden,

		consts.ErrNotConfigured: http.StatusPreconditionRequired,

		consts.ErrJwtControllerError: http.StatusInternalServerError,
	}
	for e, s := range statuses {
		if errors.Is(err, e) {
			return s
		}
	}
	return http.StatusInternalServerError
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
