package repo

import (
	"testing"
)

func Test_ContactDbSchema(t *testing.T) {
	schema := ContactSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("user schema is invalid: %v", err)
	}
}
