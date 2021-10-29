package repo

import (
	"testing"
)

func Test_TeamDbSchema(t *testing.T) {
	schema := TeamSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("team schema is invalid: %v", err)
	}
}
