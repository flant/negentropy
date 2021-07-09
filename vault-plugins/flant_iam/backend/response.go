package backend

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func responseWithData(m interface{}) (*logical.Response, error) {
	// normally, almost no sensitive is sent via HTTP
	return responseWithSensitiveData(m, false)
}

func responseWithSensitiveData(m interface{}, includeSensitive bool) (*logical.Response, error) {
	if !includeSensitive {
		m = model.OmitSensitive(m)
	}

	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = jsonutil.DecodeJSON(raw, &data)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: data,
	}

	return resp, err
}

func responseWithDataAndCode(req *logical.Request, m interface{}, status int) (*logical.Response, error) {
	resp, err := responseWithData(m)
	if err != nil {
		return nil, err
	}
	return logical.RespondWithStatusCode(resp, req, status)
}

func responseErr(req *logical.Request, err error) (*logical.Response, error) {
	switch err {
	case model.ErrNotFound:
		return responseErrMessage(req, err.Error(), http.StatusNotFound)
	case model.ErrBadVersion:
		return responseErrMessage(req, err.Error(), http.StatusConflict)
	case model.ErrBadOrigin:
		return responseErrMessage(req, err.Error(), http.StatusForbidden)
	default:
		return nil, err
	}
}

func responseErrMessage(req *logical.Request, message string, status int) (*logical.Response, error) {
	rr := logical.ErrorResponse(message)
	return logical.RespondWithStatusCode(rr, req, status)
}

// commit wraps the committing and error logging
func commit(tx *io.MemoryStoreTxn, logger log.Logger) error {
	err := tx.Commit()
	if err != nil {
		logger.Error("failed to commit", "err", err)
		return fmt.Errorf("request failed, try again")
	}
	return nil
}
