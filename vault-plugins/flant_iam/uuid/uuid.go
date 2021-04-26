package uuid

import (
	"fmt"

	"github.com/google/uuid"
)

func New() string {
	v, err := uuid.NewRandom()
	if err != nil {
		return New()
	}
	return v.String()
}

// Pattern matches UUID case-insensitively
func Pattern(name string) string {
	const (
		uuidPattern = "(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12})"
	)
	return fmt.Sprintf(`(?P<%s>%s)`, name, uuidPattern)
}

func OptionalPathParam(name string) string {
	return optional(Pattern(name)) + "$"
}

// OptionalParamRegex should be just as strict as framework.GenericNameRegex, but optional
func optional(pattern string) string {
	return fmt.Sprintf("(/%s)?", pattern)
}
