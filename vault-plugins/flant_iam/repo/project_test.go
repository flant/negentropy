package repo

import (
	"testing"
)

func Test_ProjectDbSchema(t *testing.T) {
	schema := ProjectSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("Project schema is invalid: %v", err)
	}
}
