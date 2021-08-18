package repo

import (
	"testing"
)

func Test_ServiceAccountDbSchema(t *testing.T) {
	schema := ServiceAccountSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("service account schema is invalid: %v", err)
	}
}
