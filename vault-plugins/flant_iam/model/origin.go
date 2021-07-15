package model

type ObjectOrigin string

const (
	OriginIAM ObjectOrigin = "iam"
)
func ValidateOrigin(origin ObjectOrigin) error {
	if origin == OriginIAM {
		return nil
	}
	return ErrBadOrigin
}
