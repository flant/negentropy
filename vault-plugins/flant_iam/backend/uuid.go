package backend

import (
	"fmt"

	"github.com/google/uuid"
)

func genUUID() string {
	v, err := uuid.NewRandom()
	if err != nil {
		return genUUID()
	}
	return v.String()
}

// UUIDParam matches UUID case-insensitively
func UUIDParam(name string) string {
	const (
		uuidPattern = "(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12})"
	)
	return fmt.Sprintf(`(?P<%s>%s)`, name, uuidPattern)
}

func OptionalUUIDParam(name string) string {
	return OptionalSubpattern(UUIDParam(name))+ "$"
}

/*
	(re)	        numbered capturing group (submatch)
	(?P<name>re)	named & numbered capturing group (submatch)
	(?:re)	        non-capturing group
	(?flags)	set flags within current group; non-capturing
	(?flags:re)	set flags during re; non-capturing
*/

// OptionalParamRegex should be just as strict as framework.GenericNameRegex, but optional
func OptionalSubpattern(pattern string) string {
	return fmt.Sprintf("(/%s)?", pattern)
}

type uuidGenerator struct{}

func (g *uuidGenerator) GenerateID() string {
	return genUUID()
}
