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

func responseWithSensitiveDataAndCode(req *logical.Request, m interface{}, status int) (*logical.Response, error) {
	resp, err := responseWithSensitiveData(m, true)
	if err != nil {
		return nil, err
	}
	return logical.RespondWithStatusCode(resp, req, status)
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
		return responseNotFound(req)
	case model.ErrBadVersion:
		return responseBadVersion(req)
	case model.ErrBadOrigin:
		return responseBadOrigin(req)
	default:
		return nil, err
	}
}

func responseErrMessage(req *logical.Request, message string, status int) (*logical.Response, error) {
	rr := logical.ErrorResponse(message)
	return logical.RespondWithStatusCode(rr, req, status)
}

func responseNotFound(req *logical.Request) (*logical.Response, error) {
	return responseErrMessage(req, "not found", http.StatusNotFound)
}

func responseBadVersion(req *logical.Request) (*logical.Response, error) {
	return responseErrMessage(req, "bad version", http.StatusConflict)
}

func responseBadOrigin(req *logical.Request) (*logical.Response, error) {
	return responseErrMessage(req, "bad origin", http.StatusForbidden)
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
