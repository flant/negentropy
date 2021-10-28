package repo

import (
	"testing"
)

func Test_TeamMateDbSchema(t *testing.T) {
	schema := TeammateSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("teammate schema is invalid: %v", err)
	}
}
