package model

import (
	"testing"
)

func Test_MergedSchema(t *testing.T) {
	schema, err := mergeSchema()
	if err != nil {
		t.Fatalf("cannot merge schema: %v", err)
	}
	if err := schema.Validate(); err != nil {
		t.Fatalf("merged schema is invalid: %v", err)
	}
}

func Test_NewDb(t *testing.T) {
	_, err := NewDB()
	if err != nil {
		t.Fatalf("cannot create db: %v", err)
	}
}
