package consts

import "fmt"

var (
	ErrNotFound           = fmt.Errorf("not found")
	ErrAlreadyExists      = fmt.Errorf("already exists")
	ErrBadVersion         = fmt.Errorf("bad version")
	ErrBadOrigin          = fmt.Errorf("bad origin")
	ErrBadScopeRole       = fmt.Errorf("wrong scope of role")
	ErrIsArchived         = fmt.Errorf("entity is archived")
	ErrIsNotArchived      = fmt.Errorf("entity is not archived")
	ErrNoUUID             = fmt.Errorf("uuid is required")
	ErrJwtDisabled        = fmt.Errorf("JWT is disabled")
	ErrJwtControllerError = fmt.Errorf("JWT controller error")
	ErrNilPointer         = fmt.Errorf("nil pointer passed")
	ErrWrongType          = fmt.Errorf("wrong type")
	ErrInvalidArg         = fmt.Errorf("invalid value of argument")
	ErrNotConfigured      = fmt.Errorf("not configured")
	ErrAccessForbidden    = fmt.Errorf("access forbidden")
	ErrNotHandledObject   = fmt.Errorf("type is not handled yet")
	CriticalCodeError     = fmt.Errorf("CRITICAL ERROR AT CODE")
)
