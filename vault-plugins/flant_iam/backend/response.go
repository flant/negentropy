package backend

import (
	"fmt"
	"net/http"

	"github.com/flant/negentropy/vault-plugins/shared/io"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

var (
	ErrNotFound        = fmt.Errorf("not found")
	ErrVersionMismatch = fmt.Errorf("version mismatch")
	ErrAlreadyExists   = fmt.Errorf("already exists")
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

func responseNotFound(req *logical.Request, who string) (*logical.Response, error) {
	rr := logical.ErrorResponse(who + " not found")
	return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
}

func responseVersionMismatch(req *logical.Request) (*logical.Response, error) {
	rr := logical.ErrorResponse("version mismatch")
	return logical.RespondWithStatusCode(rr, req, http.StatusConflict)
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
