package model

import (
	"testing"
)

func Test_GroupDbSchema(t *testing.T) {
	schema := GroupSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("group schema is invalid: %v", err)
	}
}
