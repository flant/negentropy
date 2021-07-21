package errors

import "errors"

type EntryNotFound struct {
	msg string
}

func NewEntryNotFound(msg string) error {
	return &EntryNotFound{msg: msg}
}

func (e *EntryNotFound) Error() string {
	return e.msg
}

func IsEntryNotFound(err error) bool {
	var e *EntryNotFound
	if errors.As(err, &e) {
		return true
	}

	return false
}
