package repo

import (
	"testing"
)

func Test_MergeSchema(t *testing.T) {
	schema, err := GetSchema()
	if err != nil {
		t.Fatalf("cannot merge schema: %v", err)
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("merged schema is invalid: %v", err)
	}
}
