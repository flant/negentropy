package backend

import (
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
)

func responseErr(req *logical.Request, err error) (*logical.Response, error) {
	switch err {
	case model.ErrNotFound:
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusNotFound)
	case model.ErrBadVersion:
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusConflict)
	default:
		return nil, err
	}
}

func nameFromRequest(d *framework.FieldData) (string, *logical.Response) {
	sourceName := d.Get("name").(string)
	if sourceName == "" {
		return "", logical.ErrorResponse("name is required")
	}

	return sourceName, nil
}
