package repo

import (
	"testing"
)

func Test_RoleBindingDbSchema(t *testing.T) {
	schema := RoleBindingSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("role binding schema is invalid: %v", err)
	}
}
