package backend

import (
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func responseErr(req *logical.Request, err error) (*logical.Response, error) {
	switch err {
	case model.ErrNotFound:
		return ResponseErrMessage(req, err.Error(), http.StatusNotFound)
	case model.ErrBadVersion:
		return ResponseErrMessage(req, err.Error(), http.StatusConflict)
	default:
		return nil, err
	}
}

func ResponseErrMessage(req *logical.Request, message string, status int) (*logical.Response, error) {
	rr := logical.ErrorResponse(message)
	return logical.RespondWithStatusCode(rr, req, status)
}

func nameFromRequest(d *framework.FieldData) (string, *logical.Response) {
	sourceName := d.Get("name").(string)
	if sourceName == "" {
		return "", logical.ErrorResponse("name is required")
	}

	return sourceName, nil
}
