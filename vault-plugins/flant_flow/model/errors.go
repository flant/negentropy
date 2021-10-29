package model

import "fmt"

var (
	ErrNotFound            = fmt.Errorf("not found")
	ErrAlreadyExists       = fmt.Errorf("already exists")
	ErrBadVersion          = fmt.Errorf("bad version")
	ErrBadOrigin           = fmt.Errorf("bad origin")
	ErrNeedSingleArgument  = fmt.Errorf("must provide only a single argument")
	ErrNeedDoubleArgument  = fmt.Errorf("must provide two arguments")
	ErrBadProjectScopeRole = fmt.Errorf("project_scoped_role is not allowed")
	ErrIsArchived          = fmt.Errorf("entity is archived")
	ErrIsNotArchived       = fmt.Errorf("entity is not archived")
)
