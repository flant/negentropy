package backend

import (
	"fmt"
	"net/http"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
)

var (
	errJwtDisabled        = fmt.Errorf("JWT is disabled")
	errJwtControllerError = fmt.Errorf("JWT controller error")
)

func isJwtEnabled(tx *io.MemoryStoreTxn, controller *jwt.Controller) error {
	isEnabled, err := controller.IsEnabled(tx)
	if err != nil {
		return errJwtControllerError
	}
	if !isEnabled {
		return errJwtDisabled
	}
	return nil
}

func responseErr(req *logical.Request, err error) (*logical.Response, error) {
	switch err {
	case model.ErrNotFound:
		return responseErrMessage(req, err.Error(), http.StatusNotFound)
	case model.ErrBadVersion:
		return responseErrMessage(req, err.Error(), http.StatusConflict)
	case model.ErrBadOrigin, errJwtDisabled:
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
