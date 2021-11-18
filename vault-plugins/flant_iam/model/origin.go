package model

import "github.com/flant/negentropy/vault-plugins/shared/consts"

type ObjectOrigin string

const (
	OriginIAM          ObjectOrigin = "iam"
	OriginServerAccess ObjectOrigin = "server_access"
)

func ValidateOrigin(origin ObjectOrigin) error {
	if origin == OriginIAM || origin == OriginServerAccess {
		return nil
	}
	return consts.ErrBadOrigin
}
