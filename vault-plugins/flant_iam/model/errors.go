package model

import "fmt"

var (
	ErrNotFound        = fmt.Errorf("not found")
	ErrVersionMismatch = fmt.Errorf("version mismatch")
	ErrAlreadyExists   = fmt.Errorf("already exists")
)
