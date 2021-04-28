package model

import (
	"testing"
)

func Test_RoleDbSchema(t *testing.T) {
	schema := RoleSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("role schema is invalid: %v", err)
	}
}
