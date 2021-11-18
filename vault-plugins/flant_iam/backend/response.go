package backend

import (
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/jwt"
)

func isJwtEnabled(tx *io.MemoryStoreTxn, controller *jwt.Controller) error {
	isEnabled, err := controller.IsEnabled(tx)
	if err != nil {
		return consts.ErrJwtControllerError
	}
	if !isEnabled {
		return consts.ErrJwtDisabled
	}
	return nil
}
