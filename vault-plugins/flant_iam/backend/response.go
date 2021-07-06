package backend

import (
	"fmt"
	"net/http"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func responseWithData(m model.Marshaller) (*logical.Response, error) {
	json, err := m.Marshal(false)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = jsonutil.DecodeJSON(json, &data)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: data,
	}

	return resp, err
}

func responseWithDataAndCode(req *logical.Request, m model.Marshaller, status int) (*logical.Response, error) {
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
	case model.ErrVersionMismatch:
		return responseVersionMismatch(req)
	case model.ErrOriginMismatch:
		return responseOriginMismatch(req)
	default:
		return nil, err
	}
}

func responseNotFound(req *logical.Request) (*logical.Response, error) {
	rr := logical.ErrorResponse("not found")
	return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
}

func responseVersionMismatch(req *logical.Request) (*logical.Response, error) {
	rr := logical.ErrorResponse("version mismatch")
	return logical.RespondWithStatusCode(rr, req, http.StatusConflict)
}

func responseOriginMismatch(req *logical.Request) (*logical.Response, error) {
	rr := logical.ErrorResponse("readonly object")
	return logical.RespondWithStatusCode(rr, req, http.StatusForbidden)
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
