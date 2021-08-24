package jwtauth

import (
	"net/http"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func responseErr(req *logical.Request, err error) (*logical.Response, error) {
	switch err {
	case model.ErrNotFound:
		return responseErrMessage(req, err.Error(), http.StatusNotFound)
	case model.ErrBadVersion:
		return responseErrMessage(req, err.Error(), http.StatusConflict)
	default:
		return nil, err
	}
}

func responseErrMessage(req *logical.Request, message string, status int) (*logical.Response, error) {
	rr := logical.ErrorResponse(message)
	return logical.RespondWithStatusCode(rr, req, status)
}
