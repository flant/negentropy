package extension_server_access

import (
	"fmt"
	"net/http"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

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
