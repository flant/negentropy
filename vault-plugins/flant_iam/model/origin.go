package model

type ObjectOrigin string

const (
	OriginIAM          ObjectOrigin = "iam"
	OriginServerAccess ObjectOrigin = "server_access"
)

func ValidateOrigin(origin ObjectOrigin) error {
	if origin == OriginIAM || origin == OriginServerAccess {
		return nil
	}
	return ErrBadOrigin
}
